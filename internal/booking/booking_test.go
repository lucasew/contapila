package booking

import (
	"math/big"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/config"
	"github.com/lucasew/contapila-go/internal/ledger"
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

func TestBooker(t *testing.T) {
	cfg, _ := config.Load([]byte(""), "test.cue")

	tests := []struct {
		name           string
		directives     []ledger.Directive
		wantErrorCount int
		wantWarnCount  int
		verifyResidual func(t *testing.T, directives []ledger.Directive)
	}{
		{
			name: "balanced simple",
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
			name: "residual assignment",
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
			verifyResidual: func(t *testing.T, directives []ledger.Directive) {
				txn := directives[2].(ledger.Transaction)
				if txn.Postings[1].Amount == nil {
					t.Errorf("residual amount not assigned")
				} else if txn.Postings[1].Amount.Number.Cmp(mustParseRat("30.00")) != 0 {
					t.Errorf("wrong residual amount: %s", txn.Postings[1].Amount.Number)
				}
			},
		},
		{
			name: "unopened account warning",
			directives: []ledger.Directive{
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("0"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 0,
			wantWarnCount:  1,
		},
		{
			name: "after close error",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Close{Date: mustParseDate("2020-01-10"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-11"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("0"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 1,
			wantWarnCount:  0,
		},
		{
			name: "tolerance check",
			directives: []ledger.Directive{
				ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
				ledger.Transaction{
					Date: mustParseDate("2020-01-02"),
					Postings: []ledger.Posting{
						{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("0.000006"), Commodity: "BRL"}},
					},
				},
			},
			wantErrorCount: 1,
			wantWarnCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBooker(cfg)
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
			if tt.verifyResidual != nil {
				tt.verifyResidual(t, tt.directives)
			}
		})
	}
}

func TestBooker_ToleranceFromConfig(t *testing.T) {
	cfg, _ := config.Load([]byte("commodities: BRL: precision: 2"), "test.cue")
	b := NewBooker(cfg)

	directives := []ledger.Directive{
		ledger.Open{Date: mustParseDate("2020-01-01"), Account: "Assets:Cash"},
		ledger.Transaction{
			Date: mustParseDate("2020-01-02"),
			Postings: []ledger.Posting{
				{Account: "Assets:Cash", Amount: &ledger.Amount{Number: mustParseRat("0.004"), Commodity: "BRL"}},
			},
		},
	}

	b.Book(directives)

	for _, d := range b.Diagnostics {
		if d.Severity == Error {
			t.Errorf("expected balanced within tolerance 0.005, but got error: %s", d.Message)
		}
	}
}
