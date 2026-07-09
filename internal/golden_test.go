package internal

import (
	"contapila/internal/engine"
	"contapila/internal/project"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type Expected struct {
	Ledgers map[string]ExpectedLedger `json:"ledgers"`
}

type ExpectedLedger struct {
	Balances map[string]map[string]string `json:"balances"`
	Errors   []string                     `json:"errors"`
	Warnings []string                     `json:"warnings"`
}

func TestGolden(t *testing.T) {
	entries, err := os.ReadDir("../testdata/golden")
	if err != nil {
		t.Fatalf("failed to read testdata/golden: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			testDir := filepath.Join("../testdata/golden", entry.Name())
			proj, err := project.Discover(testDir)
			if err != nil {
				t.Fatalf("Discover failed: %v", err)
			}

			results := make(map[string]*engine.Ledger)
			for _, li := range proj.Ledgers {
				l, err := engine.ProcessLedger(proj, li)
				if err != nil {
					t.Fatalf("ProcessLedger %s failed: %v", li.Name, err)
				}
				results[li.Name] = l
			}

			// Load expected.json
			expData, err := os.ReadFile(filepath.Join(testDir, "expected.json"))
			if err != nil {
				t.Fatalf("failed to read expected.json: %v", err)
			}

			var expected Expected
			if err := json.Unmarshal(expData, &expected); err != nil {
				t.Fatalf("failed to unmarshal expected.json: %v", err)
			}

			for name, expLedger := range expected.Ledgers {
				gotLedger, ok := results[name]
				if !ok {
					t.Errorf("expected ledger %s not found in results", name)
					continue
				}

				// Compare balances
				gotBalances := make(map[string]map[string]string)
				for acc, inv := range gotLedger.Balances {
					gotBalances[acc] = make(map[string]string)
					for comm, pos := range inv {
						gotBalances[acc][comm] = pos.Units.String()
					}
				}

				if expLedger.Balances != nil && !reflect.DeepEqual(expLedger.Balances, gotBalances) {
					t.Errorf("balances mismatch for ledger %s\nwant: %v\ngot:  %v", name, expLedger.Balances, gotBalances)
				}

				// Compare errors
				sort.Strings(gotLedger.Errors)
				sort.Strings(expLedger.Errors)
				if !reflect.DeepEqual(expLedger.Errors, gotLedger.Errors) {
					t.Errorf("errors mismatch for ledger %s\nwant: %v\ngot:  %v", name, expLedger.Errors, gotLedger.Errors)
				}

				// Compare warnings
				sort.Strings(gotLedger.Warnings)
				sort.Strings(expLedger.Warnings)
				if !reflect.DeepEqual(expLedger.Warnings, gotLedger.Warnings) {
					t.Errorf("warnings mismatch for ledger %s\nwant: %v\ngot:  %v", name, expLedger.Warnings, gotLedger.Warnings)
				}
			}
		})
	}
}
