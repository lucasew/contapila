package bootstrap

import (
	"strings"
	"testing"

	"github.com/lucasew/contapila-go/internal/ledger"
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
