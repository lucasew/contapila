package parser

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
2024-01-01 open Assets:Checking
2024-01-01 open Expenses:Food

2024-01-10 * "Grocery" "Weekly shop"
  Assets:Checking  -50.00 USD
  Expenses:Food     50.00 USD

2024-01-20 * "Investment"
  Assets:Broker    10 AAPL {150.00 USD}
  Assets:Checking -1500.00 USD

2024-01-25 * "Sell"
  Assets:Broker    -5 AAPL @@ 1000.00 USD
  Assets:Checking  1000.00 USD
  Income:Gains
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(directives) != 5 { // 2 opens + 3 txns
		t.Errorf("Expected 5 directives, got %d", len(directives))
	}

	txn3 := directives[4].(*Transaction)
	if txn3.Narration != "Sell" {
		t.Errorf("Expected narration 'Sell', got '%s'", txn3.Narration)
	}
	if len(txn3.Postings) != 3 {
		t.Errorf("Expected 3 postings, got %d", len(txn3.Postings))
	}

	p1 := txn3.Postings[0]
	if p1.Price == nil || p1.Price.Number.String() != "200.00000" {
		t.Errorf("Expected unit price 200.00000 from @@ 1000.00, got %v", p1.Price)
	}
}
