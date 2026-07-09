package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contapila-loader-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.beancount")
	otherFile := filepath.Join(tmpDir, "other.beancount")

	os.WriteFile(mainFile, []byte("include \"other.beancount\"\n2024-01-01 * \"Main\""), 0644)
	os.WriteFile(otherFile, []byte("2024-01-01 * \"Other\""), 0644)

	directives, err := LoadFiles(mainFile, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}

	if len(directives) != 2 {
		t.Errorf("expected 2 directives, got %d", len(directives))
	}
}
