package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type Project struct {
	Root string
}

func Discover(cwd string) (*Project, error) {
	curr := cwd
	for {
		marker := filepath.Join(curr, "contapila.cue")
		if _, err := os.Stat(marker); err == nil {
			return &Project{Root: curr}, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return nil, fmt.Errorf("not a contapila project (no contapila.cue found in parents of %s)", cwd)
}

func (p *Project) LedgerPath(name string) (string, error) {
	path := filepath.Join(p.Root, name, "main.beancount")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("ledger %q not found at %s", name, path)
	}
	return path, nil
}

func (p *Project) ListLedgers() ([]string, error) {
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join(p.Root, entry.Name(), "main.beancount")
			if _, err := os.Stat(path); err == nil {
				names = append(names, entry.Name())
			}
		}
	}
	return names, nil
}
