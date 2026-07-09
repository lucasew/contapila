package ledger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLedger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contapila-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create contapila.cue
	err = os.WriteFile(filepath.Join(tmpDir, "contapila.cue"), []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create personal ledger
	err = os.Mkdir(filepath.Join(tmpDir, "personal"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	beancountContent := `
2024-01-01 open Assets:Checking
2024-01-01 open Expenses:Food

2024-01-02 * "Grocery"
  Assets:Checking  -10.00 USD
  Expenses:Food     10.00 USD

2024-01-03 balance Assets:Checking -10.00 USD
`
	err = os.WriteFile(filepath.Join(tmpDir, "personal", "main.beancount"), []byte(beancountContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	project, err := DiscoverProject(filepath.Join(tmpDir, "personal"))
	if err != nil {
		t.Fatalf("DiscoverProject failed: %v", err)
	}

	ledger, err := project.LoadLedger("personal")
	if err != nil {
		t.Fatalf("LoadLedger failed: %v", err)
	}

	diagnostics, err := ledger.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(diagnostics) > 0 {
		t.Errorf("Expected 0 diagnostics, got %d: %v", len(diagnostics), diagnostics)
	}

	balances, err := ledger.GetBalances(time.Now())
	if err != nil {
		t.Fatalf("GetBalances failed: %v", err)
	}

	expectedBalances := map[string]string{
		"Assets:Checking": "-10/1",
		"Expenses:Food":    "10/1",
	}

	if len(balances) != 2 {
		t.Errorf("Expected 2 balance entries, got %d", len(balances))
	}

	for _, b := range balances {
		if expected, ok := expectedBalances[b.Account]; ok {
			if b.Amount.String() != expected {
				t.Errorf("Account %s: expected balance %s, got %s", b.Account, expected, b.Amount.String())
			}
		} else {
			t.Errorf("Unexpected balance for account %s", b.Account)
		}
	}
}

func TestLedgerErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contapila-test-errors")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "contapila.cue"), []byte(""), 0644)
	os.Mkdir(filepath.Join(tmpDir, "personal"), 0755)

	beancountContent := `
2024-01-01 open Assets:Checking
2024-01-01 open Assets:Checking ; Error: duplicate open

2024-01-02 * "Grocery"
  Assets:Cash      -10.00 USD ; Warning: unopened account
  Expenses:Food     10.00 USD ; Warning: unopened account

2024-01-03 balance Assets:Checking -1.00 USD ; Error: balance assertion failed
`
	os.WriteFile(filepath.Join(tmpDir, "personal", "main.beancount"), []byte(beancountContent), 0644)

	project, _ := DiscoverProject(tmpDir)
	ledger, _ := project.LoadLedger("personal")

	diagnostics, _ := ledger.Check()

	errCount := 0
	warnCount := 0
	for _, d := range diagnostics {
		if d.Severity == "error" {
			errCount++
		} else if d.Severity == "warn" {
			warnCount++
		}
	}

	if errCount != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", errCount, diagnostics)
	}
	if warnCount != 2 {
		t.Errorf("Expected 2 warnings, got %d: %v", warnCount, diagnostics)
	}
}
