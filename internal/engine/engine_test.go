package engine

import (
	"contapila/internal/project"
	"os"
	"path/filepath"
	"testing"
)

func TestEngine(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "contapila.cue"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmp, "personal"), 0755)

	beancount := `
2024-01-01 open Assets:Checking
2024-01-01 open Expenses:Food

2024-01-10 * "Grocery"
  Assets:Checking  -50.00 USD
  Expenses:Food

2024-01-11 balance Assets:Checking -50.00 USD
`
	ledgerPath := filepath.Join(tmp, "personal", "main.beancount")
	os.WriteFile(ledgerPath, []byte(beancount), 0644)

	proj, _ := project.Discover(tmp)
	ledger, err := ProcessLedger(proj, proj.Ledgers[0])
	if err != nil {
		t.Fatalf("ProcessLedger failed: %v", err)
	}

	if len(ledger.Errors) > 0 {
		t.Errorf("Unexpected errors: %v", ledger.Errors)
	}

	inv := ledger.Balances["Assets:Checking"]
	if inv["USD"].Units.String() != "-50.00000" {
		t.Errorf("Expected -50 USD in Checking, got %s", inv["USD"].Units)
	}
}
