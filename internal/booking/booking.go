package booking

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/config"
	"github.com/lucasew/contapila-go/internal/ledger"
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
	config       *config.Config
	accountOpen  map[string]time.Time
	accountClose map[string]time.Time
	Diagnostics  []Diagnostic
}

func NewBooker(cfg *config.Config) *Booker {
	return &Booker{
		config:       cfg,
		accountOpen:  make(map[string]time.Time),
		accountClose: make(map[string]time.Time),
	}
}

func (b *Booker) Book(directives []ledger.Directive) {
	// Sort directives by date
	sort.SliceStable(directives, func(i, j int) bool {
		di := directives[i]
		dj := directives[j]
		if di.GetDate().Equal(dj.GetDate()) {
			return getPriority(di) < getPriority(dj)
		}
		return di.GetDate().Before(dj.GetDate())
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
		}
	}
}

func getPriority(d ledger.Directive) int {
	switch d.(type) {
	case ledger.Open:
		return 0
	case ledger.Transaction:
		return 1
	case ledger.Close:
		return 2
	default:
		return 3
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

		// Assign residual amount to the elided posting
		if len(nonZeroImbalances) == 1 {
			comm := nonZeroImbalances[0]
			negated := new(big.Rat).Neg(imbalances[comm])
			t.Postings[residualPostingIdx].Amount = &ledger.Amount{
				Number:    negated,
				Commodity: comm,
			}
		} else if len(nonZeroImbalances) == 0 {
			// Transaction already balanced, residual is zero.
			// We can pick any commodity or just leave it nil, but usually Beancount-style
			// empty posting on a balanced txn is fine and means zero.
			// Let's pick the first commodity if any, or just don't assign.
		}
	} else {
		// No residual, check if all commodities balance within tolerance
		for comm, bal := range imbalances {
			tolerance := b.getTolerance(comm)
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
}

func (b *Booker) getTolerance(commodity string) *big.Rat {
	precision := 5 // Default per SPEC
	if b.config != nil {
		path := fmt.Sprintf("commodities[\"%s\"].precision", commodity)
		v := b.config.Value.LookupPath(config.ParsePath(path))
		if v.Exists() {
			if p, err := v.Int64(); err == nil {
				precision = int(p)
			}
		}
	}

	// tolerance = 0.5 * 10^-precision
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	// tolerance = 1 / (2 * denom)
	return new(big.Rat).SetFrac(big.NewInt(5), new(big.Int).Mul(denom, big.NewInt(10)))
}
