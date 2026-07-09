package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmp := t.TempDir()

	// Create structure
	// tmp/contapila.cue
	// tmp/personal/main.beancount
	// tmp/empresa/main.beancount
	// tmp/prices.beancount

	os.WriteFile(filepath.Join(tmp, "contapila.cue"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmp, "prices.beancount"), []byte(""), 0644)

	os.Mkdir(filepath.Join(tmp, "personal"), 0755)
	os.WriteFile(filepath.Join(tmp, "personal", "main.beancount"), []byte(""), 0644)

	os.Mkdir(filepath.Join(tmp, "empresa"), 0755)
	os.WriteFile(filepath.Join(tmp, "empresa", "main.beancount"), []byte(""), 0644)

	// Test from inside a ledger dir
	p, err := Discover(filepath.Join(tmp, "personal"))
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if p.Root != tmp {
		t.Errorf("Expected root %s, got %s", tmp, p.Root)
	}

	if len(p.Ledgers) != 2 {
		t.Errorf("Expected 2 ledgers, got %d", len(p.Ledgers))
	}

	if p.Prices == "" {
		t.Errorf("Expected prices.beancount to be found")
	}
}
