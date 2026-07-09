package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type Project struct {
	Root    string
	Ledgers []LedgerInfo
	Prices  string // path to prices.beancount
}

type LedgerInfo struct {
	Name string
	Path string // path to main.beancount
}

func Discover(startPath string) (*Project, error) {
	root, err := findRoot(startPath)
	if err != nil {
		return nil, err
	}

	p := &Project{
		Root: root,
	}

	// Look for prices.beancount
	pricesPath := filepath.Join(root, "prices.beancount")
	if _, err := os.Stat(pricesPath); err == nil {
		p.Prices = pricesPath
	}

	// Scan for ledgers: root/*/main.beancount
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ledgerPath := filepath.Join(root, entry.Name(), "main.beancount")
		if _, err := os.Stat(ledgerPath); err == nil {
			p.Ledgers = append(p.Ledgers, LedgerInfo{
				Name: entry.Name(),
				Path: ledgerPath,
			})
		}
	}

	return p, nil
}

func findRoot(startPath string) (string, error) {
	curr, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(curr, "contapila.cue")); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("not a contapila project (contapila.cue not found in any parent directory)")
		}
		curr = parent
	}
}
