package parser

import (
	"contapila/internal/model"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
option "operating_currency" "USD"
include "other.beancount"

2024-01-01 price AAPL 150.00 USD

2024-01-02 * "Buy AAPL"
  Assets:Broker  10 AAPL {150.00 USD}
  Assets:Cash
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	foundOption := false
	foundInclude := false
	foundPrice := false
	foundTxn := false

	for _, d := range directives {
		switch v := d.(type) {
		case *model.Option:
			if v.Name == "operating_currency" && v.Value == "USD" {
				foundOption = true
			}
		case *model.Include:
			if v.Path == "other.beancount" {
				foundInclude = true
			}
		case *model.PriceDirective:
			if v.Price.Commodity == "AAPL" && v.Price.Value.FloatString(2) == "150.00" {
				foundPrice = true
			}
		case *model.Transaction:
			if v.Narration == "Buy AAPL" {
				foundTxn = true
				if len(v.Postings) != 2 {
					t.Errorf("expected 2 postings, got %d", len(v.Postings))
				}
				p0 := v.Postings[0]
				if p0.Account != "Assets:Broker" || p0.Units.Currency != "AAPL" || p0.Cost.Value.FloatString(2) != "150.00" {
					t.Errorf("unexpected posting 0: %+v", p0)
				}
			}
		}
	}

	if !foundOption { t.Error("missing option") }
	if !foundInclude { t.Error("missing include") }
	if !foundPrice { t.Error("missing price") }
	if !foundTxn { t.Error("missing txn") }
}
