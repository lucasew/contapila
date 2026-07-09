package parser

import (
	"github.com/lucasew/contapila-go/internal/ledger"
	"math/big"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
2020-01-01 open Assets:Cash
2020-01-01 open Expenses:Food

2020-01-02 * "Lunch"
  Assets:Cash       -30.00 BRL
  Expenses:Food
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(directives) != 3 {
		t.Fatalf("got %d directives, want 3", len(directives))
	}

	if _, ok := directives[0].(ledger.Open); !ok {
		t.Errorf("directive 0 is not Open, got %T", directives[0])
	}

	txn, ok := directives[2].(ledger.Transaction)
	if !ok {
		t.Fatalf("directive 2 is not Transaction, got %T", directives[2])
	}

	if len(txn.Postings) != 2 {
		t.Errorf("got %d postings, want 2", len(txn.Postings))
	}

	if txn.Postings[0].Amount == nil {
		t.Errorf("posting 0 amount should not be nil")
	}

	if txn.Postings[1].Amount != nil {
		t.Errorf("posting 1 amount should be nil (residual)")
	}
}

func TestParseBalanceAndPad(t *testing.T) {
	input := `
2020-01-01 pad Assets:Cash Equity:Opening-Balances
2020-01-02 balance Assets:Cash 100.00 BRL
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(directives) != 2 {
		t.Fatalf("got %d directives, want 2", len(directives))
	}

	pad, ok := directives[0].(ledger.Pad)
	if !ok {
		t.Fatalf("directive 0 is not Pad, got %T", directives[0])
	}
	if pad.Account != "Assets:Cash" || pad.SourceAccount != "Equity:Opening-Balances" {
		t.Errorf("incorrect pad directive: %+v", pad)
	}

	bal, ok := directives[1].(ledger.Balance)
	if !ok {
		t.Fatalf("directive 1 is not Balance, got %T", directives[1])
	}
	if bal.Account != "Assets:Cash" || bal.Amount.Commodity != "BRL" {
		t.Errorf("incorrect balance directive: %+v", bal)
	}
	if bal.Amount.Number.Cmp(new(big.Rat).SetInt64(100)) != 0 {
		t.Errorf("incorrect balance amount: %s", bal.Amount.Number.String())
	}
}
