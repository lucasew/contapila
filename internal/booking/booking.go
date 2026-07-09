package booking

import (
	"contapila/internal/ledger"
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

type Booker struct {
	accountOpen     map[string]time.Time
	accountClose    map[string]time.Time
	accountBalances map[string]map[string]*big.Rat // account -> commodity -> amount
	activePads      map[string]ledger.Pad          // account -> latest pad directive
	Diagnostics     []Diagnostic
}

func NewBooker() *Booker {
	return &Booker{
		accountOpen:     make(map[string]time.Time),
		accountClose:    make(map[string]time.Time),
		accountBalances: make(map[string]map[string]*big.Rat),
		activePads:      make(map[string]ledger.Pad),
	}
}

func typeOrder(d ledger.Directive) int {
	switch d.(type) {
	case ledger.Open:
		return 0
	case ledger.Close:
		return 1
	case ledger.Pad:
		return 2
	case ledger.Transaction:
		return 3
	case ledger.Balance:
		return 4
	default:
		return 5
	}
}

func (b *Booker) Book(directives []ledger.Directive) {
	// Sort directives by date
	sort.SliceStable(directives, func(i, j int) bool {
		if !directives[i].GetDate().Equal(directives[j].GetDate()) {
			return directives[i].GetDate().Before(directives[j].GetDate())
		}
		return typeOrder(directives[i]) < typeOrder(directives[j])
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
		case ledger.Pad:
			b.activePads[v.Account] = v
		case ledger.Balance:
			b.handleBalance(v)
		case ledger.Transaction:
			b.bookTransaction(v)
		}
	}
}

func (b *Booker) updateBalance(account, commodity string, amount *big.Rat) {
	if b.accountBalances[account] == nil {
		b.accountBalances[account] = make(map[string]*big.Rat)
	}
	if b.accountBalances[account][commodity] == nil {
		b.accountBalances[account][commodity] = new(big.Rat)
	}
	b.accountBalances[account][commodity].Add(b.accountBalances[account][commodity], amount)
}

func (b *Booker) getAccountBalance(account, commodity string) *big.Rat {
	if b.accountBalances[account] == nil {
		return new(big.Rat)
	}
	if b.accountBalances[account][commodity] == nil {
		return new(big.Rat)
	}
	return b.accountBalances[account][commodity]
}

func (b *Booker) handleBalance(bal ledger.Balance) {
	current := b.getAccountBalance(bal.Account, bal.Amount.Commodity)

	// Check if there's an active pad
	if pad, ok := b.activePads[bal.Account]; ok {
		diff := new(big.Rat).Sub(bal.Amount.Number, current)
		if diff.Sign() != 0 {
			// Insert padding transaction
			paddingTxn := ledger.Transaction{
				Date:      pad.Date,
				Narration: fmt.Sprintf("Padding for balance of %s %s", bal.Amount.Number.FloatString(2), bal.Amount.Commodity),
				Postings: []ledger.Posting{
					{Account: bal.Account, Amount: &ledger.Amount{Number: diff, Commodity: bal.Amount.Commodity}},
					{Account: pad.SourceAccount, Amount: &ledger.Amount{Number: new(big.Rat).Neg(diff), Commodity: bal.Amount.Commodity}},
				},
			}
			b.bookTransaction(paddingTxn)
			// Update current after padding
			current = b.getAccountBalance(bal.Account, bal.Amount.Commodity)
		}
		delete(b.activePads, bal.Account) // Pad is used
	}

	// Now check balance
	tolerance := big.NewRat(5, 1000000) // Default tolerance
	diff := new(big.Rat).Sub(bal.Amount.Number, current)
	if new(big.Rat).Abs(diff).Cmp(tolerance) > 0 {
		b.Diagnostics = append(b.Diagnostics, Diagnostic{
			Severity: Error,
			Message:  fmt.Sprintf("Balance failed for %s: expected %s %s, got %s %s (diff %s)", bal.Account, bal.Amount.Number.FloatString(2), bal.Amount.Commodity, current.FloatString(2), bal.Amount.Commodity, diff.FloatString(2)),
			Date:     bal.Date,
		})
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
			val := new(big.Rat).Neg(imbalances[comm])
			b.updateBalance(t.Postings[residualPostingIdx].Account, comm, val)
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

	// Update balance for non-residual postings
	for i, p := range t.Postings {
		if i == residualPostingIdx {
			continue
		}
		if p.Amount != nil {
			b.updateBalance(p.Account, p.Amount.Commodity, p.Amount.Number)
		}
	}
}
