package ledger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/contapila/contapila/internal/parser"
)

func TestLoader(t *testing.T) {
	tmp := t.TempDir()

	mainPath := filepath.Join(tmp, "main.beancount")
	os.WriteFile(mainPath, []byte(`
include "sub.beancount"
include "glob/*.beancount"
2024-01-01 commodity MAIN
`), 0644)

	os.WriteFile(filepath.Join(tmp, "sub.beancount"), []byte(`
2024-01-01 commodity SUB
`), 0644)

	err := os.Mkdir(filepath.Join(tmp, "glob"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(tmp, "glob", "1.beancount"), []byte(`
2024-01-01 commodity GLOB1
`), 0644)
	os.WriteFile(filepath.Join(tmp, "glob", "2.beancount"), []byte(`
2024-01-01 commodity GLOB2
`), 0644)

	loader := NewLoader()
	l, err := loader.Load(mainPath, tmp)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	directives := l.Directives

	expected := []string{"SUB", "GLOB1", "GLOB2", "MAIN"}
	// Directives include includes that are expanded but the original includes are replaced by their content
	// Actually my implementation appends content and keeps all directives including the include itself if I wasn't careful?
	// Wait, l.loadRecursive appends content for include but doesn't keep the include itself.

	if len(directives) != len(expected) {
		t.Fatalf("Expected %d directives, got %d", len(expected), len(directives))
	}

	for i, name := range expected {
		c, ok := directives[i].(parser.Commodity)
		if !ok || c.Name != name {
			t.Errorf("%d: expected commodity %s, got %v", i, name, directives[i])
		}
	}
}

func TestLoaderCycle(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.beancount")
	b := filepath.Join(tmp, "b.beancount")

	os.WriteFile(a, []byte(`include "b.beancount"`), 0644)
	os.WriteFile(b, []byte(`include "a.beancount"`), 0644)

	loader := NewLoader()
	_, err := loader.Load(a, tmp)
	if err == nil {
		t.Fatal("Expected error for include cycle, got nil")
	}
}

func TestLoaderDedupe(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.beancount")
	sub := filepath.Join(tmp, "sub.beancount")

	os.WriteFile(main, []byte(`
include "sub.beancount"
include "sub.beancount"
`), 0644)
	os.WriteFile(sub, []byte(`2024-01-01 commodity SUB`), 0644)

	loader := NewLoader()
	l, err := loader.Load(main, tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(l.Directives) != 1 {
		t.Errorf("Expected 1 directive (deduped), got %d", len(l.Directives))
	}
}

func TestLoaderMissing(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.beancount")
	os.WriteFile(main, []byte(`include "missing.beancount"`), 0644)

	loader := NewLoader()
	_, err := loader.Load(main, tmp)
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}
