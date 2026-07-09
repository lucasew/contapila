package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "proj")
	err := os.MkdirAll(filepath.Join(root, "ledger1"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(root, "contapila.cue"), []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(root, "ledger1", "main.beancount"), []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(root, "ledger1")
	p, err := Discover(sub)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if p.Root != root {
		t.Errorf("Expected root %s, got %s", root, p.Root)
	}

	lp, err := p.LedgerPath("ledger1")
	if err != nil {
		t.Errorf("LedgerPath failed: %v", err)
	}
	expectedLp := filepath.Join(root, "ledger1", "main.beancount")
	if lp != expectedLp {
		t.Errorf("Expected ledger path %s, got %s", expectedLp, lp)
	}

	ledgers, err := p.ListLedgers()
	if err != nil {
		t.Errorf("ListLedgers failed: %v", err)
	}
	if len(ledgers) != 1 || ledgers[0] != "ledger1" {
		t.Errorf("Expected [ledger1], got %v", ledgers)
	}
}
