package adapter

import (
	"math/big"
	"testing"
	"time"

	"github.com/contapila/contapila/internal/directive"
)

func TestParse_Unavailable(t *testing.T) {
	_, _, err := Parse("test.beancount", []byte(""))
	if err != ErrParserUnavailable {
		t.Errorf("expected ErrParserUnavailable, got %v", err)
	}
}

// TestDirectiveTypes verifies that we can construct all MVP directive types.
// This ensures the internal/directive package has enough types to stub for later issues.
func TestDirectiveTypes(t *testing.T) {
	now := time.Now()
	rat100 := big.NewRat(100, 1)

	directives := []directive.Directive{
		&directive.Option{Name: "operating_currency", Value: "BRL"},
		&directive.Include{Path: "others.beancount"},
		&directive.Commodity{Date: now, Currency: "BRL"},
		&directive.Open{Date: now, Account: "Assets:Checking", Currencies: []string{"BRL"}},
		&directive.Close{Date: now, Account: "Assets:Checking"},
		&directive.Transaction{
			Date:      now,
			Flag:      "*",
			Payee:     "Supermarket",
			Narration: "Weekly groceries",
			Postings: []directive.Posting{
				{
					Account: "Assets:Checking",
					Amount: &directive.Amount{
						Number:    rat100,
						Commodity: "BRL",
					},
				},
				{
					Account: "Expenses:Food",
				},
			},
		},
		&directive.MarketPrice{
			Date:      now,
			Commodity: "AAPL",
			Amount: directive.Amount{
				Number:    big.NewRat(150, 1),
				Commodity: "USD",
			},
		},
		&directive.Balance{
			Date:    now,
			Account: "Assets:Checking",
			Amount: directive.Amount{
				Number:    big.NewRat(900, 1),
				Commodity: "BRL",
			},
		},
		&directive.Pad{
			Date:      now,
			Account:   "Assets:Checking",
			SourceAcc: "Equity:OpeningBalances",
		},
		&directive.Note{
			Date:    now,
			Account: "Assets:Checking",
			Comment: "Checked balance",
		},
		&directive.Event{
			Date:  now,
			Type:  "type",
			Value: "value",
		},
		&directive.Unknown{
			Type: "custom",
			Text: "some custom directive",
		},
	}

	if len(directives) != 12 {
		t.Errorf("expected 12 directives, got %d", len(directives))
	}
}
