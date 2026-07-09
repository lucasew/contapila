package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type Project struct {
	Root    string
	Ledgers map[string]string // Name -> Path to main.beancount
}

// FindRoot walks upward from startDir looking for contapila.cue
func FindRoot(startDir string) (string, error) {
	curr := startDir
	for {
		marker := filepath.Join(curr, "contapila.cue")
		if _, err := os.Stat(marker); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("not a contapila project (no contapila.cue found in parents)")
		}
		curr = parent
	}
}

// LoadProject identifies ledgers and other project components
func LoadProject(root string) (*Project, error) {
	p := &Project{
		Root:    root,
		Ledgers: make(map[string]string),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			mainPath := filepath.Join(root, entry.Name(), "main.beancount")
			if _, err := os.Stat(mainPath); err == nil {
				p.Ledgers[entry.Name()] = mainPath
			}
		}
	}

	return p, nil
}
