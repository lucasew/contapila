package parser

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
option "operating_currency" "BRL"
include "path/to/file.beancount"

2024-01-01 commodity PETR4
2024-01-01 open Assets:Bank:Checking BRL,USD
2024-01-01 open Liabilities:CreditCard
2024-02-01 close Liabilities:CreditCard

2024-03-01 * "Buy"
  Assets:Bank:Checking  -100 BRL
  Assets:Investments     100 BRL
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(directives) != 7 {
		t.Errorf("Expected 7 directives, got %d", len(directives))
	}

	check := func(i int, expected Directive) {
		if i >= len(directives) {
			t.Fatalf("Directive %d not found", i)
		}
		got := directives[i]
		switch e := expected.(type) {
		case Option:
			g, ok := got.(Option)
			if !ok || g != e {
				t.Errorf("%d: expected %v, got %v", i, e, got)
			}
		case Include:
			g, ok := got.(Include)
			if !ok || g != e {
				t.Errorf("%d: expected %v, got %v", i, e, got)
			}
		case Commodity:
			g, ok := got.(Commodity)
			if !ok || g != e {
				t.Errorf("%d: expected %v, got %v", i, e, got)
			}
		case Open:
			g, ok := got.(Open)
			if !ok || g.Account != e.Account || strings.Join(g.Currencies, ",") != strings.Join(e.Currencies, ",") {
				t.Errorf("%d: expected %v, got %v", i, e, got)
			}
		case Close:
			g, ok := got.(Close)
			if !ok || g != e {
				t.Errorf("%d: expected %v, got %v", i, e, got)
			}
		case Transaction:
			_, ok := got.(Transaction)
			if !ok {
				t.Errorf("%d: expected Transaction, got %v", i, got)
			}
		}
	}

	check(0, Option{Name: "operating_currency", Value: "BRL"})
	check(1, Include{Path: "path/to/file.beancount"})
	check(2, Commodity{Name: "PETR4"})
	check(3, Open{Account: "Assets:Bank:Checking", Currencies: []string{"BRL", "USD"}})
	check(4, Open{Account: "Liabilities:CreditCard"})
	check(5, Close{Account: "Liabilities:CreditCard"})

	txn := directives[6].(*Transaction)
	if len(txn.Postings) != 2 {
		t.Errorf("Expected 2 postings, got %d", len(txn.Postings))
	}
}

func TestParseDashes(t *testing.T) {
	input := `2024-01-01 open Assets:Bank-Accounts:Checking`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(directives) != 1 {
		t.Fatalf("Expected 1 directive, got %d", len(directives))
	}
	o, ok := directives[0].(Open)
	if !ok || o.Account != "Assets:Bank-Accounts:Checking" {
		t.Errorf("Expected account with dashes, got %v", directives[0])
	}
}
