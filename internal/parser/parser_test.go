package parser

import (
	"testing"

	"github.com/lucasew/contapila-go/internal/ast"
)

func TestParseBasic(t *testing.T) {
	src := []byte(`
2020-01-01 open Assets:Cash
2020-01-01 open Expenses:Food
2020-01-02 * "Lunch"
  Assets:Cash  -30.00 BRL
  Expenses:Food
`)
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags.HasErrors() {
		t.Fatalf("diags: %v", diags)
	}
	var opens, txns int
	for _, d := range dirs {
		switch d.(type) {
		case ast.Open:
			opens++
		case ast.Transaction:
			txns++
		}
	}
	if opens != 2 || txns != 1 {
		t.Fatalf("opens=%d txns=%d dirs=%d", opens, txns, len(dirs))
	}
}

func TestParsePayeeAndNarration(t *testing.T) {
	src := []byte(`
2020-01-01 open Assets:Cash
2020-01-01 open Expenses:Food
2020-01-05 * "Restaurant Foo" "Lunch"
  Assets:Cash  -30.00 BRL
  Expenses:Food
2020-01-06 * "Coffee only narration"
  Assets:Cash  -10.00 BRL
  Expenses:Food
`)
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags.HasErrors() {
		t.Fatalf("diags: %v", diags)
	}
	var both, narrOnly *ast.Transaction
	for _, d := range dirs {
		txn, ok := d.(ast.Transaction)
		if !ok {
			continue
		}
		switch txn.Date.Format("2006-01-02") {
		case "2020-01-05":
			t := txn
			both = &t
		case "2020-01-06":
			t := txn
			narrOnly = &t
		}
	}
	if both == nil || narrOnly == nil {
		t.Fatalf("missing txns both=%v narrOnly=%v", both, narrOnly)
	}
	if both.Payee != "Restaurant Foo" || both.Narration != "Lunch" {
		t.Fatalf("payee+narration: payee=%q narration=%q", both.Payee, both.Narration)
	}
	if narrOnly.Payee != "" || narrOnly.Narration != "Coffee only narration" {
		t.Fatalf("narration-only: payee=%q narration=%q", narrOnly.Payee, narrOnly.Narration)
	}
}
