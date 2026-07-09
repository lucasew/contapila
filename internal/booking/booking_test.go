package booking

import (
	"github.com/lucasew/contapila-go/internal/ledger"
	"math/big"
	"testing"
	"time"
)

func mustParseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func mustParseRat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("invalid rat: " + s)
	}
	return r
}

func TestBooker_Book(t *testing.T) {
	tests := []struct {
		name           string
		directives     []ledger.Directive
		wantErrorCount int
		wantWarnCount  int
	}{
		{
			name: "balanced simple transaction",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Expenses:Food"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("-30.00"), Commodity: "BRL"}},
						{Account: "Expenses:Food", Amount: &ledger.Amount{Number: mustParseRat("30.00"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  0,
		},
		{
			name: "balanced with residual",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Expenses:Food"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("-30.00"), Commodity: "BRL"}},
						{Account: "Expenses:Food"},
					},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  0,
		},
		{
			name: "unbalanced no residual",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("-30.00"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 1,
			wantWarnCount:  0,
		},
		{
			name: "unopened account warning",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("-30.00"), Commodity: "BRL"}},
						{Account: "Expenses:Misc", Amount: &ledger.Amount{Number: mustParseRat("30.00"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  1,
		},
		{
			name: "posting after close",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Close{Date: mustParseDate("2020-01-02"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-03"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("-30.00"), Commodity: "BRL"}},
						{Account: "Expenses:Misc", Amount: &ledger.Amount{Number: mustParseRat("30.00"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 1,
			wantWarnCount:  1, // Expenses:Misc not opened
		},
		{
			name: "successful balance assertion",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("100"), Commodity: "BRL"}},
						{Account: "Equity:Opening-Balances"},
					},
				},
				ledger.Balance{
					Date:    mustParseDate("2020-01-03"),
					Account: "Assets:Cash",
					Amount:  ledger.Amount{Number: mustParseRat("100"), Commodity: "BRL"},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  1, // Equity:Opening-Balances not opened
		},
		{
			name: "failed balance assertion",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("100"), Commodity: "BRL"}},
						{Account: "Equity:Opening-Balances"},
					},
				},
				ledger.Balance{
					Date:    mustParseDate("2020-01-03"),
					Account: "Assets:Cash",
					Amount:  ledger.Amount{Number: mustParseRat("150"), Commodity: "BRL"},
				},
			},
			wantErrorCount: 1,
			wantWarnCount:  1,
		},
		{
			name: "successful pad and balance",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Equity:Opening-Balances"},
				ledger.Pad{
					Date:          mustParseDate("2020-01-02"),
					Account:       "Assets:Cash",
					SourceAccount: "Equity:Opening-Balances",
				},
				ledger.Balance{
					Date:    mustParseDate("2020-01-03"),
					Account: "Assets:Cash",
					Amount:  ledger.Amount{Number: mustParseRat("100"), Commodity: "BRL"},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBooker()
			b.Book(tt.directives)

			errCount := 0
			warnCount := 0
			for _, d := range b.Diagnostics {
				if d.Severity == Error {
					errCount++
				} else if d.Severity == Warning {
					warnCount++
				}
			}

			if errCount != tt.wantErrorCount {
				t.Errorf("got %d errors, want %d. Diagnostics: %v", errCount, tt.wantErrorCount, b.Diagnostics)
			}
			if warnCount != tt.wantWarnCount {
				t.Errorf("got %d warnings, want %d. Diagnostics: %v", warnCount, tt.wantWarnCount, b.Diagnostics)
			}
		})
	}
}
