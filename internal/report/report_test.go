package report

import (
	"github.com/lucasew/contapila-go/internal/ledger"
	"math/big"
	"testing"
	"time"
)

func TestReportMath(t *testing.T) {
	d1 := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)

	directives := []ledger.Directive{
		ledger.Open{Date: d1, Account: "Assets:Cash"},
		ledger.Open{Date: d1, Account: "Expenses:Food"},
		ledger.Transaction{
			Date: d1,
			Postings: []ledger.Posting{
				{Account: "Assets:Cash", Amount: &ledger.Amount{Number: big.NewRat(100, 1), Commodity: "USD"}},
				{Account: "Income:Salary", Amount: &ledger.Amount{Number: big.NewRat(-100, 1), Commodity: "USD"}},
			},
		},
		ledger.Transaction{
			Date: d2,
			Postings: []ledger.Posting{
				{Account: "Assets:Cash", Amount: &ledger.Amount{Number: big.NewRat(-30, 1), Commodity: "USD"}},
				{Account: "Expenses:Food", Amount: &ledger.Amount{Number: big.NewRat(30, 1), Commodity: "USD"}},
			},
		},
	}

	t.Run("Balances", func(t *testing.T) {
		res := Balances(directives, d2)
		foundCash := false
		for _, r := range res {
			if r.Account == "Assets:Cash" {
				foundCash = true
				if r.Units.Cmp(big.NewRat(70, 1)) != 0 {
					t.Errorf("got cash %v, want 70", r.Units)
				}
			}
		}
		if !foundCash {
			t.Error("Assets:Cash not found in balances")
		}

		resAugust := Balances(directives, d3)
		if len(resAugust) == 0 {
			t.Error("expected balances in August")
		}
	})

	t.Run("PnL", func(t *testing.T) {
		var txns []ledger.Transaction
		for _, d := range directives {
			if txn, ok := d.(ledger.Transaction); ok {
				txns = append(txns, txn)
			}
		}

		res := PnL(txns, d1, d2)
		foundSalary := false
		foundFood := false
		for _, r := range res {
			if r.Account == "Income:Salary" {
				foundSalary = true
				if r.Amount.Cmp(big.NewRat(-100, 1)) != 0 {
					t.Errorf("got salary %v, want -100", r.Amount)
				}
			}
			if r.Account == "Expenses:Food" {
				foundFood = true
				if r.Amount.Cmp(big.NewRat(30, 1)) != 0 {
					t.Errorf("got food %v, want 30", r.Amount)
				}
			}
		}
		if !foundSalary || !foundFood {
			t.Errorf("missing PnL entries: salary=%v, food=%v", foundSalary, foundFood)
		}
	})
}
