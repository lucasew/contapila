package project

import (
	"fmt"
	"os"
	"path/filepath"
)

const MarkerFile = "contapila.cue"

func FindRoot(startDir string) (string, error) {
	curr := startDir
	for {
		marker := filepath.Join(curr, MarkerFile)
		if _, err := os.Stat(marker); err == nil {
			return curr, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", fmt.Errorf("not a contapila project: %s not found in parent directories", MarkerFile)
}

type LedgerInfo struct {
	Name string
	Path string
}

func DiscoverLedgers(root string) ([]LedgerInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var ledgers []LedgerInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mainBeancount := filepath.Join(root, entry.Name(), "main.beancount")
		if _, err := os.Stat(mainBeancount); err == nil {
			ledgers = append(ledgers, LedgerInfo{
				Name: entry.Name(),
				Path: mainBeancount,
			})
		}
	}
	return ledgers, nil
}
