package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lucasew/contapila-go/internal/config"
	"github.com/lucasew/contapila-go/internal/prices"
)

type Ledger struct {
	Name     string
	MainPath string
}

type Project struct {
	Root          string
	Config        *config.Config
	Ledgers       []Ledger
	PricesPath    string
	PricesMissing bool
	PricesEmpty   bool
}

const ProjectMarker = "contapila.cue"
const PricesFilename = "prices.beancount"
const LedgerEntrypoint = "main.beancount"

func findRoot(startDir string) (string, error) {
	curr, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		markerPath := filepath.Join(curr, ProjectMarker)
		if _, err := os.Stat(markerPath); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return "", fmt.Errorf("not a contapila project (searched upward for %s)", ProjectMarker)
}

func discoverLedgers(root string) ([]Ledger, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var ledgers []Ledger
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ledgerPath := filepath.Join(root, entry.Name(), LedgerEntrypoint)
		if _, err := os.Stat(ledgerPath); err == nil {
			ledgers = append(ledgers, Ledger{
				Name:     entry.Name(),
				MainPath: ledgerPath,
			})
		}
	}

	return ledgers, nil
}

func OpenProject(cwd string) (*Project, error) {
	root, err := findRoot(cwd)
	if err != nil {
		return nil, err
	}

	// Discover ledgers first so CUE can type #LedgerName from the filesystem.
	ledgers, err := discoverLedgers(root)
	if err != nil {
		return nil, fmt.Errorf("failed to discover ledgers: %w", err)
	}
	discovered := make([]config.Ledger, 0, len(ledgers))
	for _, l := range ledgers {
		discovered = append(discovered, config.Ledger{Name: l.Name, Main: l.MainPath})
	}

	pricesPath := filepath.Join(root, PricesFilename)
	pricesMissing := false
	pricesEmpty := false
	var pricePairs []config.PricePair
	if info, err := os.Stat(pricesPath); os.IsNotExist(err) {
		slog.Warn("prices.beancount is missing", "path", pricesPath)
		pricesMissing = true
	} else if err == nil {
		if info.Size() == 0 {
			slog.Warn("prices.beancount is empty", "path", pricesPath)
			pricesEmpty = true
		} else {
			// Pair inventory only for CUE (not full series). Full DB loads in engine.OpenProject.
			if pdb, _, err := prices.LoadFile(pricesPath); err != nil {
				slog.Warn("failed loading prices for CUE pair inject", "err", err)
			} else {
				for _, p := range pdb.Pairs() {
					pricePairs = append(pricePairs, config.PricePair{Base: p.Base, Quote: p.Quote})
				}
			}
		}
	} else {
		return nil, fmt.Errorf("failed to stat prices file: %w", err)
	}

	cuePath := filepath.Join(root, ProjectMarker)
	cueBytes, err := os.ReadFile(cuePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ProjectMarker, err)
	}

	cfg, err := config.Load(cueBytes, cuePath, discovered, pricePairs)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Project{
		Root:          root,
		Config:        cfg,
		Ledgers:       ledgers,
		PricesPath:    pricesPath,
		PricesMissing: pricesMissing,
		PricesEmpty:   pricesEmpty,
	}, nil
}
