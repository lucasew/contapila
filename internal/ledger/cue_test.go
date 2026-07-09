package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderCUE(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.beancount")
	os.WriteFile(main, []byte(`
option "operating_currency" "BRL"
2024-01-01 commodity PETR4
2024-01-01 open Assets:Bank:Checking
`), 0644)

	os.WriteFile(filepath.Join(tmp, "contapila.cue"), []byte(`
commodities: PETR4: precision: 2
`), 0644)

	loader := NewLoader()
	l, err := loader.Load(main, tmp)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(l.Config.OperatingCurrency) != 1 || l.Config.OperatingCurrency[0] != "BRL" {
		t.Errorf("Expected OperatingCurrency [BRL], got %v", l.Config.OperatingCurrency)
	}

	p, ok := l.Config.Commodities["PETR4"]
	if !ok || p.Precision != 2 {
		t.Errorf("Expected PETR4 precision 2, got %v", l.Config.Commodities["PETR4"])
	}

	// Default precision from prelude
	p2, ok := l.Config.Commodities["USD"]
	// Wait, USD is not in facts, so it won't be in Commodities map unless it's in CUE or directives
	// Let's add it to facts via directive
	os.WriteFile(main, []byte(`
option "operating_currency" "BRL"
2024-01-01 commodity PETR4
2024-01-01 commodity USD
2024-01-01 open Assets:Bank:Checking
`), 0644)

	l, _ = loader.Load(main, tmp)
	p2, ok = l.Config.Commodities["USD"]
	if !ok || p2.Precision != 5 {
		t.Errorf("Expected USD precision 5 (default), got %v", l.Config.Commodities["USD"])
	}

	acc, ok := l.Config.Accounts["Assets:Bank:Checking"]
	if !ok || !acc.Opened {
		t.Errorf("Expected Assets:Bank:Checking opened, got %v", l.Config.Accounts["Assets:Bank:Checking"])
	}
}

func TestLoaderClose(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.beancount")
	os.WriteFile(main, []byte(`
2024-01-01 open Assets:Bank:Checking BRL
2024-02-01 close Assets:Bank:Checking
`), 0644)

	loader := NewLoader()
	l, err := loader.Load(main, tmp)
	if err != nil {
		t.Fatal(err)
	}

	acc := l.Config.Accounts["Assets:Bank:Checking"]
	if !acc.Opened || !acc.Closed {
		t.Errorf("Expected opened and closed, got %+v", acc)
	}
	if len(acc.Currencies) != 1 || acc.Currencies[0] != "BRL" {
		t.Errorf("Expected currencies [BRL], got %v", acc.Currencies)
	}
}

func TestLoaderCUEConflict(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.beancount")
	os.WriteFile(main, []byte(`
2024-01-01 commodity PETR4
`), 0644)

	os.WriteFile(filepath.Join(tmp, "contapila.cue"), []byte(`
commodities: PETR4: precision: "not-an-int"
`), 0644)

	loader := NewLoader()
	_, err := loader.Load(main, tmp)
	if err == nil {
		t.Fatal("Expected CUE conflict error, got nil")
	}
}
