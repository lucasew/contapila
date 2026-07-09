package booking

import (
	"github.com/lucasew/contapila-go/internal/ledger"
	"fmt"
	"math/big"
	"sort"
	"time"
)

type Severity int

const (
	Warning Severity = iota
	Error
)

type Diagnostic struct {
	Severity Severity
	Message  string
	Date     time.Time
}

type Position struct {
	Units *big.Rat
	Cost  *big.Rat // Total cost for these units
}

type Booker struct {
	accountOpen  map[string]time.Time
	accountClose map[string]time.Time
	Diagnostics  []Diagnostic
	// Account -> Commodity -> Position
	Balances     map[string]map[string]*Position
	Transactions []ledger.Transaction
}

func NewBooker() *Booker {
	return &Booker{
		accountOpen:  make(map[string]time.Time),
		accountClose: make(map[string]time.Time),
		Balances:     make(map[string]map[string]*Position),
	}
}

func (b *Booker) Book(directives []ledger.Directive) {
	// Sort directives by date
	sort.SliceStable(directives, func(i, j int) bool {
		return directives[i].GetDate().Before(directives[j].GetDate())
	})

	for _, d := range directives {
		switch v := d.(type) {
		case ledger.Open:
			if _, ok := b.accountOpen[v.Account]; ok {
				b.Diagnostics = append(b.Diagnostics, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("Account already opened: %s", v.Account),
					Date:     v.Date,
				})
			}
			b.accountOpen[v.Account] = v.Date
		case ledger.Close:
			if _, ok := b.accountClose[v.Account]; ok {
				b.Diagnostics = append(b.Diagnostics, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("Account already closed: %s", v.Account),
					Date:     v.Date,
				})
			}
			b.accountClose[v.Account] = v.Date
		case ledger.Transaction:
			b.bookTransaction(v)
			b.Transactions = append(b.Transactions, v)
		}
	}
}

func (b *Booker) bookTransaction(t ledger.Transaction) {
	var residualPostingIdx = -1
	imbalances := make(map[string]*big.Rat)

	for i, p := range t.Postings {
		// Check account status
		openDate, opened := b.accountOpen[p.Account]
		if !opened {
			b.Diagnostics = append(b.Diagnostics, Diagnostic{
				Severity: Warning,
				Message:  fmt.Sprintf("Account not opened: %s", p.Account),
				Date:     t.Date,
			})
		} else if t.Date.Before(openDate) {
			b.Diagnostics = append(b.Diagnostics, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("Transaction date %s is before account %s open date %s", t.Date.Format("2006-01-02"), p.Account, openDate.Format("2006-01-02")),
				Date:     t.Date,
			})
		}

		if closeDate, closed := b.accountClose[p.Account]; closed && !t.Date.Before(closeDate) {
			b.Diagnostics = append(b.Diagnostics, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("Transaction date %s is at or after account %s close date %s", t.Date.Format("2006-01-02"), p.Account, closeDate.Format("2006-01-02")),
				Date:     t.Date,
			})
		}

		if p.Amount == nil {
			if residualPostingIdx != -1 {
				b.Diagnostics = append(b.Diagnostics, Diagnostic{
					Severity: Error,
					Message:  "Transaction has more than one residual posting",
					Date:     t.Date,
				})
				return
			}
			residualPostingIdx = i
		} else {
			if imbalances[p.Amount.Commodity] == nil {
				imbalances[p.Amount.Commodity] = new(big.Rat)
			}
			imbalances[p.Amount.Commodity].Add(imbalances[p.Amount.Commodity], p.Amount.Number)
		}
	}

	if residualPostingIdx != -1 {
		// Fill residual posting
		// If there are multiple commodities with imbalances, this might be tricky.
		// Beancount only allows one elided posting and it must balance EVERYTHING.
		// If multiple commodities are involved, they must all balance to zero if possible?
		// Actually Beancount's rule: if one posting is elided, it's assigned whatever is needed to balance.
		// If there are multiple commodities, the elided posting would need to have multiple amounts?
		// "at most one posting with missing amount to absorb the residual"
		// "assign it the negated residual per commodity as required so the txn balances"
		// This implies the residual posting can have multiple commodities if necessary,
		// but usually it's just one.

		// Wait, ledger.Posting only has one Amount.
		// If I have:
		// 2020-01-01 *
		//   Assets:A   10 USD
		//   Assets:B   20 EUR
		//   Expenses:Misc
		// The Expenses:Misc needs -10 USD and -20 EUR.
		// My data structure for Posting only has one Amount.
		// SPEC says: "assign it the negated residual per commodity as required so the txn balances"

		// If there's only one commodity in imbalances, it's easy.
		// If there's more than one, we might need multiple postings or a different structure.
		// But Beancount allows ONLY ONE elided posting.

		// Re-reading SPEC §7.4: "At most one posting with missing amount absorbs the remainder (typically gains)."

		// Let's see how I should handle multiple commodities.
		// "If residual exists but still cannot balance (e.g. two commodities both residual-needed with one empty posting — handle as Beancount does or error clearly) → error with a clear message."

		// If there are multiple commodities with non-zero imbalances, one posting cannot balance them all if it can only have one amount.
		// Unless the residual posting is SPLIT into multiple postings?
		// No, usually it means the transaction is invalid if it requires multiple commodities to balance but only one is elided.

		var nonZeroImbalances []string
		for comm, bal := range imbalances {
			if bal.Sign() != 0 {
				nonZeroImbalances = append(nonZeroImbalances, comm)
			}
		}

		if len(nonZeroImbalances) > 1 {
			b.Diagnostics = append(b.Diagnostics, Diagnostic{
				Severity: Error,
				Message:  fmt.Sprintf("Transaction requires residual balancing for multiple commodities (%v) but only one residual posting exists", nonZeroImbalances),
				Date:     t.Date,
			})
			return
		}

		// If len(nonZeroImbalances) == 0, the residual is zero.
		// If len(nonZeroImbalances) == 1, we assign the negated imbalance.
		if len(nonZeroImbalances) == 1 {
			comm := nonZeroImbalances[0]
			negated := new(big.Rat).Neg(imbalances[comm])
			t.Postings[residualPostingIdx].Amount = &ledger.Amount{
				Number:    negated,
				Commodity: comm,
			}
		}
	} else {
		// No residual, check if all commodities balance within tolerance
		tolerance := big.NewRat(5, 1000000) // 0.000005
		for comm, bal := range imbalances {
			absBal := new(big.Rat).Abs(bal)
			if absBal.Cmp(tolerance) > 0 {
				b.Diagnostics = append(b.Diagnostics, Diagnostic{
					Severity: Error,
					Message:  fmt.Sprintf("Transaction unbalanced for commodity %s: %s (tolerance %s)", comm, bal.FloatString(10), tolerance.FloatString(10)),
					Date:     t.Date,
				})
			}
		}
	}

	// Update balances
	for _, p := range t.Postings {
		if p.Amount == nil {
			continue
		}
		if b.Balances[p.Account] == nil {
			b.Balances[p.Account] = make(map[string]*Position)
		}
		pos := b.Balances[p.Account][p.Amount.Commodity]
		if pos == nil {
			pos = &Position{
				Units: new(big.Rat),
				Cost:  new(big.Rat),
			}
			b.Balances[p.Account][p.Amount.Commodity] = pos
		}
		pos.Units.Add(pos.Units, p.Amount.Number)
		// For now, we don't have cost tracking implemented in the parser, so cost matches units for simple cash txns.
		// Real Model A cost tracking comes in #5.
	}
}
