package ledger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/contapila/contapila/internal/cueutil"
	"github.com/contapila/contapila/internal/parser"
)

type LoadedLedger struct {
	Config       RuntimeConfig
	Directives   []parser.Directive
	Transactions []parser.Transaction
}

type Loader struct {
	seen map[string]bool
}

func NewLoader() *Loader {
	return &Loader{
		seen: make(map[string]bool),
	}
}

func (l *Loader) Load(entryPath string, projectRoot string) (*LoadedLedger, error) {
	l.seen = make(map[string]bool)
	directives, err := l.loadRecursive(entryPath, nil)
	if err != nil {
		return nil, err
	}

	facts := map[string]any{
		"operating_currency": []string{},
		"commodities":        map[string]any{},
		"accounts":           map[string]any{},
	}

	var txns []parser.Transaction
	var filtered []parser.Directive

	knownOptions := map[string]bool{
		"operating_currency": true,
	}

	for _, d := range directives {
		switch v := d.(type) {
		case parser.Option:
			if v.Name == "operating_currency" {
				facts["operating_currency"] = append(facts["operating_currency"].([]string), v.Value)
			} else if !knownOptions[v.Name] {
				slog.Warn("Unknown option", "name", v.Name)
			}
		case parser.Commodity:
			c := facts["commodities"].(map[string]any)
			if _, ok := c[v.Name]; !ok {
				c[v.Name] = map[string]any{}
			}
		case parser.Open:
			a := facts["accounts"].(map[string]any)
			if _, ok := a[v.Account]; ok {
				return nil, fmt.Errorf("duplicate open same account: %s", v.Account)
			}
			a[v.Account] = map[string]any{
				"opened":     true,
				"currencies": v.Currencies,
			}
		case parser.Close:
			a := facts["accounts"].(map[string]any)
			acc, ok := a[v.Account].(map[string]any)
			if !ok {
				acc = map[string]any{}
				a[v.Account] = acc
			}
			acc["closed"] = true
		case parser.Transaction:
			txns = append(txns, v)
		}
		filtered = append(filtered, d)
	}


	unifier := cueutil.NewUnifier()
	schemas := []string{Prelude}

	cuePath := filepath.Join(projectRoot, "contapila.cue")
	if _, err := os.Stat(cuePath); err == nil {
		content, err := os.ReadFile(cuePath)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, string(content))
	}

	v, err := unifier.Unify(schemas, []any{facts})
	if err != nil {
		return nil, err
	}

	// Ensure defaults are applied
	v, _ = v.Default()

	var cfg RuntimeConfig
	if err := unifier.Decode(v, &cfg); err != nil {
		return nil, err
	}

	return &LoadedLedger{
		Config:       cfg,
		Directives:   filtered,
		Transactions: txns,
	}, nil
}

func (l *Loader) loadRecursive(path string, stack []string) ([]parser.Directive, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Cycle detection
	for _, s := range stack {
		if s == absPath {
			return nil, fmt.Errorf("include cycle detected: %s", absPath)
		}
	}

	// Deduplication
	if l.seen[absPath] {
		return nil, nil
	}
	l.seen[absPath] = true

	directives, err := parser.ParseFile(absPath)
	if err != nil {
		return nil, err
	}

	var expanded []parser.Directive
	dir := filepath.Dir(absPath)

	for _, d := range directives {
		if inc, ok := d.(parser.Include); ok {
			incPath := inc.Path
			if !filepath.IsAbs(incPath) {
				incPath = filepath.Join(dir, incPath)
			}

			matches, err := filepath.Glob(incPath)
			if err != nil {
				return nil, fmt.Errorf("invalid include pattern %q: %w", inc.Path, err)
			}

			if len(matches) == 0 {
				if hasMeta(incPath) {
					slog.Warn("include glob matched zero files", "pattern", inc.Path, "from", absPath)
				} else {
					return nil, fmt.Errorf("include file not found: %s (from %s)", inc.Path, absPath)
				}
				continue
			}

			newStack := append(stack, absPath)
			for _, match := range matches {
				sub, err := l.loadRecursive(match, newStack)
				if err != nil {
					return nil, err
				}
				expanded = append(expanded, sub...)
			}
		} else {
			expanded = append(expanded, d)
		}
	}

	return expanded, nil
}

func hasMeta(path string) bool {
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '*', '?', '[', '\\':
			return true
		}
	}
	return false
}
