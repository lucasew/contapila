package report

import (
	"github.com/lucasew/contapila-go/internal/booking"
	"github.com/lucasew/contapila-go/internal/ledger"
	"math/big"
	"strings"
	"time"
)

type BalanceResult struct {
	Account   string
	Commodity string
	Units     *big.Rat
}

func Balances(directives []ledger.Directive, asOf time.Time) []BalanceResult {
	b := booking.NewBooker()
	var filtered []ledger.Directive
	for _, d := range directives {
		if !d.GetDate().After(asOf) {
			filtered = append(filtered, d)
		}
	}
	b.Book(filtered)

	var results []BalanceResult
	for acc, comms := range b.Balances {
		for comm, pos := range comms {
			if pos.Units.Sign() != 0 {
				results = append(results, BalanceResult{
					Account:   acc,
					Commodity: comm,
					Units:     pos.Units,
				})
			}
		}
	}
	return results
}

func Journal(transactions []ledger.Transaction, from, to time.Time) []ledger.Transaction {
	var filtered []ledger.Transaction
	for _, t := range transactions {
		if (from.IsZero() || !t.Date.Before(from)) && (to.IsZero() || !t.Date.After(to)) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

type PnLResult struct {
	Account   string
	Commodity string
	Amount    *big.Rat
}

func PnL(transactions []ledger.Transaction, from, to time.Time) []PnLResult {
	totals := make(map[string]map[string]*big.Rat) // account -> commodity -> total

	for _, t := range transactions {
		if (from.IsZero() || !t.Date.Before(from)) && (to.IsZero() || !t.Date.After(to)) {
			for _, p := range t.Postings {
				if strings.HasPrefix(p.Account, "Income:") || strings.HasPrefix(p.Account, "Expenses:") {
					if totals[p.Account] == nil {
						totals[p.Account] = make(map[string]*big.Rat)
					}
					if totals[p.Account][p.Amount.Commodity] == nil {
						totals[p.Account][p.Amount.Commodity] = new(big.Rat)
					}
					totals[p.Account][p.Amount.Commodity].Add(totals[p.Account][p.Amount.Commodity], p.Amount.Number)
				}
			}
		}
	}

	var results []PnLResult
	for acc, comms := range totals {
		for comm, val := range comms {
			results = append(results, PnLResult{
				Account:   acc,
				Commodity: comm,
				Amount:    val,
			})
		}
	}
	return results
}
