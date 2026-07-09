package parser

import (
	"contapila/internal/model"
	"os"
	"path/filepath"
)

func LoadFiles(path string, seen map[string]bool) ([]model.Directive, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if seen[absPath] {
		return nil, nil // Dedupe/Cycle prevention
	}
	seen[absPath] = true

	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	directives, err := Parse(f)
	if err != nil {
		return nil, err
	}

	var allDirectives []model.Directive
	baseDir := filepath.Dir(absPath)

	for _, d := range directives {
		if inc, ok := d.(*model.Include); ok {
			pattern := inc.Path
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(baseDir, pattern)
			}
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			for _, match := range matches {
				subDirectives, err := LoadFiles(match, seen)
				if err != nil {
					return nil, err
				}
				allDirectives = append(allDirectives, subDirectives...)
			}
		} else {
			allDirectives = append(allDirectives, d)
		}
	}

	return allDirectives, nil
}
