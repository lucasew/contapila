package engine

import (
	"math/big"
	"sort"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/booking"
	"github.com/lucasew/contapila-go/internal/period"
)

// SeriesPoint is one chart sample in operating currency.
type SeriesPoint struct {
	Date  time.Time
	Value *big.Rat // op currency
}

// BarPoint is one income/expense bin for diverging bar charts.
type BarPoint struct {
	Start   time.Time
	End     time.Time
	Label   string
	Income  *big.Rat // positive, op currency
	Expense *big.Rat // positive magnitude, op currency (chart draws down)
}

// NetWorthSeries returns net worth in op currency after each event that changes
// asset/liability positions within [from, to]. Empty bounds = open side.
// Single forward booking pass (not rebook-per-point).
func (l *Ledger) NetWorthSeries(from, to time.Time) ([]SeriesPoint, error) {
	if l.OpCurrency == "" {
		return nil, nil
	}
	return l.walkBalanceSeries(from, to, func(acct string) bool {
		return booking.IsAsset(acct) || booking.IsLiability(acct)
	}, func(b *booking.Engine, asOf time.Time) *big.Rat {
		return l.netWorthFromBook(b, asOf)
	})
}

// AccountSeries returns market value in op currency after each event touching account.
func (l *Ledger) AccountSeries(account string, from, to time.Time) ([]SeriesPoint, error) {
	if l.OpCurrency == "" || account == "" {
		return nil, nil
	}
	return l.walkBalanceSeries(from, to, func(acct string) bool {
		return AccountMatches(acct, account)
	}, func(b *booking.Engine, asOf time.Time) *big.Rat {
		return l.accountValueFromBook(b, account, asOf)
	})
}

func (l *Ledger) walkBalanceSeries(
	from, to time.Time,
	touch func(string) bool,
	value func(*booking.Engine, time.Time) *big.Rat,
) ([]SeriesPoint, error) {
	// Group directives by calendar day (stable within day via original order).
	type dayBatch struct {
		date time.Time
		dirs []ast.Directive
	}
	byDay := map[time.Time][]ast.Directive{}
	var order []time.Time
	for _, d := range l.Dirs {
		dt := d.GetDate()
		if dt.IsZero() {
			// options etc. — book at start
			dt = time.Time{}
		} else {
			dt = dateOnly(dt)
		}
		if _, ok := byDay[dt]; !ok {
			order = append(order, dt)
		}
		byDay[dt] = append(byDay[dt], d)
	}
	sort.SliceStable(order, func(i, j int) bool {
		// undated directives (options, …) book first
		if order[i].IsZero() != order[j].IsZero() {
			return order[i].IsZero()
		}
		return order[i].Before(order[j])
	})

	b := booking.New()
	var out []SeriesPoint
	for _, day := range order {
		dirs := byDay[day]
		b.Book(dirs)
		if day.IsZero() {
			continue
		}
		if !from.IsZero() && day.Before(dateOnly(from)) {
			continue
		}
		if !to.IsZero() && day.After(dateOnly(to)) {
			continue
		}
		// emit if this day could change relevant balances (txn, pad, or balance+pad)
		hit := false
		for _, d := range dirs {
			switch v := d.(type) {
			case ast.Transaction:
				for _, p := range v.Postings {
					if touch(p.Account) {
						hit = true
						break
					}
				}
			case ast.Pad:
				if touch(v.Account) || touch(v.FromAccount) {
					hit = true
				}
			case ast.Balance:
				// may apply pending pad into the account
				if touch(v.Account) {
					hit = true
				}
			}
			if hit {
				break
			}
		}
		if !hit {
			continue
		}
		out = append(out, SeriesPoint{Date: day, Value: value(b, day)})
	}
	return out, nil
}

func (l *Ledger) netWorthFromBook(b *booking.Engine, asOf time.Time) *big.Rat {
	total := big.NewRat(0, 1)
	for acct, m := range b.AllBalances() {
		if !booking.IsAsset(acct) && !booking.IsLiability(acct) {
			continue
		}
		for comm, units := range m {
			if units.Sign() == 0 {
				continue
			}
			// Natural Beancount signs (liabilities are typically negative).
			val, _ := l.convert(b, acct, comm, units, asOf)
			total.Add(total, val)
		}
	}
	return total
}

func (l *Ledger) accountValueFromBook(b *booking.Engine, account string, asOf time.Time) *big.Rat {
	total := big.NewRat(0, 1)
	for acct, m := range b.AllBalances() {
		if !AccountMatches(acct, account) {
			continue
		}
		for comm, units := range m {
			if units.Sign() == 0 {
				continue
			}
			val, _ := l.convert(b, acct, comm, units, asOf)
			total.Add(total, val)
		}
	}
	return total
}

// PnLBars returns diverging income/expense bars for bins derived from the filter.
func (l *Ledger) PnLBars(from, to time.Time, kind period.BinKind) []BarPoint {
	if from.IsZero() || to.IsZero() {
		if len(l.Book.Txns) == 0 {
			return nil
		}
		if from.IsZero() {
			from = l.Book.Txns[0].Txn.Date
			for _, bt := range l.Book.Txns {
				if bt.Txn.Date.Before(from) {
					from = bt.Txn.Date
				}
			}
		}
		if to.IsZero() {
			to = l.Book.Txns[0].Txn.Date
			for _, bt := range l.Book.Txns {
				if bt.Txn.Date.After(to) {
					to = bt.Txn.Date
				}
			}
		}
	}
	r := period.Range{Start: from, End: to}
	bins := period.IterateBins(r, kind)
	out := make([]BarPoint, 0, len(bins))
	for _, b := range bins {
		inc := big.NewRat(0, 1)
		exp := big.NewRat(0, 1)
		for _, bt := range l.Book.Txns {
			d := bt.Txn.Date
			if d.Before(b.Start) || d.After(b.End) {
				continue
			}
			for _, p := range bt.Postings {
				if p.Units == nil || p.Units.Number == nil {
					continue
				}
				val, _ := l.convertUnits(p.Units.Commodity, p.Units.Number, d)
				if booking.IsIncome(p.Account) {
					if val.Sign() < 0 {
						inc.Add(inc, new(big.Rat).Neg(val))
					} else {
						inc.Add(inc, val)
					}
				}
				if booking.IsExpense(p.Account) {
					if val.Sign() > 0 {
						exp.Add(exp, val)
					} else {
						exp.Add(exp, new(big.Rat).Neg(val))
					}
				}
			}
		}
		out = append(out, BarPoint{
			Start: b.Start, End: b.End, Label: binLabel(b, kind),
			Income: inc, Expense: exp,
		})
	}
	return out
}

func binLabel(b period.Range, kind period.BinKind) string {
	switch kind {
	case period.BinDay:
		return b.Start.Format("2006-01-02")
	case period.BinWeek:
		return b.Start.Format("2006-01-02")
	case period.BinYear:
		return b.Start.Format("2006")
	default:
		return b.Start.Format("2006-01")
	}
}

// convertUnits converts a unit amount of commodity to op currency as-of date (no inventory cost).
func (l *Ledger) convertUnits(comm string, units *big.Rat, asOf time.Time) (*big.Rat, bool) {
	if comm == l.OpCurrency || l.OpCurrency == "" {
		return new(big.Rat).Set(units), false
	}
	if rate, _, ok := l.Prices.Rate(comm, l.OpCurrency, asOf); ok {
		return new(big.Rat).Mul(new(big.Rat).Set(units), rate), false
	}
	return big.NewRat(0, 1), true
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
