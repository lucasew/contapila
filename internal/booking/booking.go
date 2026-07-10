package booking

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
)

// Position is average-cost inventory for one account+commodity (model A).
type Position struct {
	Units     *big.Rat
	TotalCost *big.Rat // in CostCommodity
	CostComm  string
}

func (p *Position) Avg() *big.Rat {
	if p.Units.Sign() == 0 {
		return big.NewRat(0, 1)
	}
	return new(big.Rat).Quo(new(big.Rat).Set(p.TotalCost), new(big.Rat).Set(p.Units))
}

// Engine books directives and tracks inventory / diagnostics.
type Engine struct {
	// account -> commodity -> position
	Inv map[string]map[string]*Position
	// bare commodity balances without cost (cash etc.) also in Inv with CostComm ""

	Open  map[string]time.Time
	Close map[string]time.Time

	// running balance per account+commodity (units only, for balance assertions)
	Bal map[string]map[string]*big.Rat

	// pad pending: account -> fromAccount (last pad wins until used)
	Pad map[string]string

	Txns   []BookedTxn
	Notes  []ast.Note
	Events []ast.Event

	Diags diag.List

	// tolerance default
	Tolerance *big.Rat
}

type BookedTxn struct {
	Txn      ast.Transaction
	Postings []FilledPosting
}

type FilledPosting struct {
	Account string
	Units   *ast.Amount // always filled after booking
	// CostBasis is total cost removed/added in cost commodity for inventory moves
	CostBasis *ast.Amount
}

func New() *Engine {
	return &Engine{
		Inv:       map[string]map[string]*Position{},
		Open:      map[string]time.Time{},
		Close:     map[string]time.Time{},
		Bal:       map[string]map[string]*big.Rat{},
		Pad:       map[string]string{},
		Tolerance: big.NewRat(5, 1000000), // 5e-6 default (precision 5 half-ulp-ish)
	}
}

func (e *Engine) Book(dirs []ast.Directive) {
	// stable sort by date; keep original order for same date via stable
	indexed := make([]ast.Directive, len(dirs))
	copy(indexed, dirs)
	sort.SliceStable(indexed, func(i, j int) bool {
		return indexed[i].GetDate().Before(indexed[j].GetDate())
	})

	for _, d := range indexed {
		switch v := d.(type) {
		case ast.Open:
			e.bookOpen(v)
		case ast.Close:
			e.bookClose(v)
		case ast.Transaction:
			e.bookTxn(v)
		case ast.Balance:
			e.bookBalance(v)
		case ast.Pad:
			e.Pad[v.Account] = v.FromAccount
		case ast.Note:
			e.Notes = append(e.Notes, v)
		case ast.Event:
			e.Events = append(e.Events, v)
		case ast.Option, ast.Commodity, ast.Price, ast.Include, ast.Document, ast.Unknown:
			// handled elsewhere
		}
	}
}

func (e *Engine) bookOpen(o ast.Open) {
	if prev, ok := e.Open[o.Account]; ok {
		e.Diags.Error(o.File, o.Line, fmt.Sprintf("duplicate open for %s (first %s)", o.Account, prev.Format("2006-01-02")))
		return
	}
	e.Open[o.Account] = o.Date
}

func (e *Engine) bookClose(c ast.Close) {
	e.Close[c.Account] = c.Date
}

func (e *Engine) checkAccount(date time.Time, account, file string, line int) {
	if od, ok := e.Open[account]; !ok {
		e.Diags.Warn(file, line, fmt.Sprintf("account not opened: %s", account))
	} else if date.Before(od) {
		e.Diags.Error(file, line, fmt.Sprintf("transaction before open of %s", account))
	}
	if cd, ok := e.Close[account]; ok && !date.Before(cd) {
		e.Diags.Error(file, line, fmt.Sprintf("posting after close of %s", account))
	}
}

func (e *Engine) bookTxn(t ast.Transaction) {
	// residual index
	resIdx := -1
	type weight struct {
		comm string
		amt  *big.Rat
	}
	weights := map[string]*big.Rat{}
	addW := func(comm string, n *big.Rat) {
		if weights[comm] == nil {
			weights[comm] = big.NewRat(0, 1)
		}
		weights[comm].Add(weights[comm], n)
	}

	filled := make([]FilledPosting, len(t.Postings))
	// First pass: inventory effects and weights for non-residual
	for i, p := range t.Postings {
		e.checkAccount(t.Date, p.Account, t.File, t.Line)
		if p.Units == nil {
			if resIdx >= 0 {
				e.Diags.Error(t.File, t.Line, "multiple residual postings in transaction")
				return
			}
			resIdx = i
			filled[i] = FilledPosting{Account: p.Account}
			continue
		}
		units := new(big.Rat).Set(p.Units.Number)
		comm := p.Units.Commodity
		fp := FilledPosting{
			Account: p.Account,
			Units:   &ast.Amount{Number: units, Commodity: comm},
		}

		// Inventory with cost (model A) when cost present OR reducing position with average
		hasInv := p.Cost != nil || e.hasPosition(p.Account, comm)
		if p.Cost != nil || (units.Sign() < 0 && e.hasPosition(p.Account, comm)) {
			hasInv = true
		}

		if hasInv && (p.Cost != nil || units.Sign() != 0) {
			if units.Sign() > 0 {
				// buy: need explicit cost
				if p.Cost == nil || p.Cost.Empty || p.Cost.Number == nil {
					e.Diags.Error(t.File, t.Line, fmt.Sprintf("buy of %s %s requires explicit cost", units.FloatString(4), comm))
					return
				}
				unitCost := p.Cost.Number
				costComm := p.Cost.Commodity
				total := new(big.Rat).Mul(new(big.Rat).Set(units), new(big.Rat).Set(unitCost))
				if err := e.buy(p.Account, comm, units, total, costComm); err != nil {
					e.Diags.Error(t.File, t.Line, err.Error())
					return
				}
				fp.CostBasis = &ast.Amount{Number: total, Commodity: costComm}
			} else if units.Sign() < 0 {
				// sell / reduce
				pos := e.getPos(p.Account, comm)
				if pos == nil || pos.Units.Sign() == 0 {
					// warn (not error): check may still pass; do not invent inventory
					e.Diags.Warn(t.File, t.Line, fmt.Sprintf("oversell %s: no inventory", comm))
					return
				}
				sellUnits := new(big.Rat).Neg(units) // positive
				if sellUnits.Cmp(pos.Units) > 0 {
					e.Diags.Warn(t.File, t.Line, fmt.Sprintf("oversell %s: have %s need %s", comm, pos.Units.FloatString(6), sellUnits.FloatString(6)))
					return
				}
				avg := pos.Avg()
				if p.Cost != nil && !p.Cost.Empty && p.Cost.Number != nil {
					// must match average
					diff := new(big.Rat).Sub(new(big.Rat).Set(p.Cost.Number), avg)
					if diff.Abs(diff).Cmp(e.Tolerance) > 0 {
						e.Diags.Error(t.File, t.Line, fmt.Sprintf("sell cost %s != average %s", p.Cost.Number.FloatString(6), avg.FloatString(6)))
						return
					}
				}
				totalCost := new(big.Rat).Mul(sellUnits, avg)
				costComm := pos.CostComm
				if err := e.sell(p.Account, comm, sellUnits, totalCost); err != nil {
					e.Diags.Error(t.File, t.Line, err.Error())
					return
				}
				fp.CostBasis = &ast.Amount{Number: new(big.Rat).Neg(totalCost), Commodity: costComm}
			}
		}

		// weight for balancing
		w := postingWeight(p, units, fp.CostBasis)
		addW(w.comm, w.amt)

		// update bare balance units
		e.addBal(p.Account, comm, units)
		filled[i] = fp
	}

	// Residual
	if resIdx >= 0 {
		// residual absorbs -sum(weights) per commodity; if >1 commodity with residual needed, error
		var nonzero []string
		for c, a := range weights {
			if a.Sign() != 0 && a.Cmp(e.Tolerance) != 0 && new(big.Rat).Abs(a).Cmp(e.Tolerance) > 0 {
				nonzero = append(nonzero, c)
			}
		}
		sort.Strings(nonzero)
		if len(nonzero) > 1 {
			e.Diags.Error(t.File, t.Line, fmt.Sprintf("residual cannot balance multiple commodities %v", nonzero))
			return
		}
		if len(nonzero) == 1 {
			c := nonzero[0]
			n := new(big.Rat).Neg(weights[c])
			filled[resIdx].Units = &ast.Amount{Number: n, Commodity: c}
			e.addBal(filled[resIdx].Account, c, n)
			addW(c, n)
		} else {
			// zero residual — leave units 0 in first weight commodity or empty
			filled[resIdx].Units = &ast.Amount{Number: big.NewRat(0, 1), Commodity: ""}
		}
	} else {
		// must balance
		for c, a := range weights {
			if new(big.Rat).Abs(a).Cmp(e.Tolerance) > 0 {
				e.Diags.Error(t.File, t.Line, fmt.Sprintf("unbalanced transaction for %s: %s", c, a.FloatString(8)))
			}
		}
	}

	e.Txns = append(e.Txns, BookedTxn{Txn: t, Postings: filled})
}

type wpair struct {
	comm string
	amt  *big.Rat
}

func postingWeight(p ast.Posting, units *big.Rat, costBasis *ast.Amount) wpair {
	// Contapila model A: inventory legs contribute cost basis to the balance so the
	// residual (Income:Gains) absorbs proceeds − cost. Cash legs use explicit amounts.
	// Price (@/@@) is for the cash/proceeds side of the txn, not inventory weight.
	if costBasis != nil && costBasis.Number != nil && costBasis.Commodity != "" {
		return wpair{costBasis.Commodity, new(big.Rat).Set(costBasis.Number)}
	}
	if p.Cost != nil && !p.Cost.Empty && p.Cost.Number != nil {
		w := new(big.Rat).Mul(new(big.Rat).Set(units), new(big.Rat).Set(p.Cost.Number))
		return wpair{p.Cost.Commodity, w}
	}
	if p.Price != nil && p.Price.Number != nil {
		var w *big.Rat
		if p.Price.Total {
			w = new(big.Rat).Abs(p.Price.Number)
			if units.Sign() < 0 {
				w.Neg(w)
			}
		} else {
			w = new(big.Rat).Mul(new(big.Rat).Set(units), new(big.Rat).Set(p.Price.Number))
		}
		return wpair{p.Price.Commodity, w}
	}
	return wpair{p.Units.Commodity, new(big.Rat).Set(units)}
}

func (e *Engine) hasPosition(account, comm string) bool {
	m := e.Inv[account]
	if m == nil {
		return false
	}
	p := m[comm]
	return p != nil && p.Units.Sign() != 0
}

func (e *Engine) getPos(account, comm string) *Position {
	m := e.Inv[account]
	if m == nil {
		return nil
	}
	return m[comm]
}

func (e *Engine) buy(account, comm string, units, totalCost *big.Rat, costComm string) error {
	if e.Inv[account] == nil {
		e.Inv[account] = map[string]*Position{}
	}
	pos := e.Inv[account][comm]
	if pos == nil {
		pos = &Position{Units: big.NewRat(0, 1), TotalCost: big.NewRat(0, 1), CostComm: costComm}
		e.Inv[account][comm] = pos
	}
	if pos.CostComm != "" && pos.CostComm != costComm && pos.Units.Sign() != 0 {
		return fmt.Errorf("cost commodity mismatch on %s %s", account, comm)
	}
	pos.CostComm = costComm
	pos.Units.Add(pos.Units, units)
	pos.TotalCost.Add(pos.TotalCost, totalCost)
	return nil
}

func (e *Engine) sell(account, comm string, sellUnits, totalCost *big.Rat) error {
	pos := e.getPos(account, comm)
	if pos == nil {
		return fmt.Errorf("no position")
	}
	pos.Units.Sub(pos.Units, sellUnits)
	pos.TotalCost.Sub(pos.TotalCost, totalCost)
	if pos.Units.Sign() == 0 {
		pos.TotalCost = big.NewRat(0, 1)
	}
	return nil
}

func (e *Engine) addBal(account, comm string, units *big.Rat) {
	if e.Bal[account] == nil {
		e.Bal[account] = map[string]*big.Rat{}
	}
	if e.Bal[account][comm] == nil {
		e.Bal[account][comm] = big.NewRat(0, 1)
	}
	e.Bal[account][comm].Add(e.Bal[account][comm], units)
}

func (e *Engine) balOf(account, comm string) *big.Rat {
	if e.Bal[account] == nil || e.Bal[account][comm] == nil {
		return big.NewRat(0, 1)
	}
	return new(big.Rat).Set(e.Bal[account][comm])
}

func (e *Engine) bookBalance(b ast.Balance) {
	actual := e.balOf(b.Account, b.Amount.Commodity)
	expected := b.Amount.Number
	diff := new(big.Rat).Sub(new(big.Rat).Set(expected), actual)
	if new(big.Rat).Abs(diff).Cmp(e.Tolerance) <= 0 {
		return
	}
	// try pad
	if from, ok := e.Pad[b.Account]; ok {
		// insert balancing: from -> account for diff
		e.checkAccount(b.Date, from, b.File, b.Line)
		e.addBal(b.Account, b.Amount.Commodity, diff)
		e.addBal(from, b.Amount.Commodity, new(big.Rat).Neg(diff))
		// also synth txn for journal
		e.Txns = append(e.Txns, BookedTxn{
			Txn: ast.Transaction{
				Meta:      ast.Meta{Date: b.Date, File: b.File, Line: b.Line},
				Flag:      "P",
				Narration: "pad",
			},
			Postings: []FilledPosting{
				{Account: b.Account, Units: &ast.Amount{Number: diff, Commodity: b.Amount.Commodity}},
				{Account: from, Units: &ast.Amount{Number: new(big.Rat).Neg(diff), Commodity: b.Amount.Commodity}},
			},
		})
		delete(e.Pad, b.Account)
		actual = e.balOf(b.Account, b.Amount.Commodity)
		diff = new(big.Rat).Sub(new(big.Rat).Set(expected), actual)
		if new(big.Rat).Abs(diff).Cmp(e.Tolerance) <= 0 {
			return
		}
	}
	e.Diags.Error(b.File, b.Line, fmt.Sprintf("balance failed %s: expected %s %s, got %s",
		b.Account, expected.FloatString(6), b.Amount.Commodity, actual.FloatString(6)))
}

// BalancesAsOf returns account+commodity units for txns on or before asOf.
// Note: Engine currently books all at once; for as-of we re-book in report layer if needed.
// Here we expose current Bal after full book.
func (e *Engine) AllBalances() map[string]map[string]*big.Rat {
	out := map[string]map[string]*big.Rat{}
	for acct, m := range e.Bal {
		out[acct] = map[string]*big.Rat{}
		for c, n := range m {
			if n.Sign() != 0 {
				out[acct][c] = new(big.Rat).Set(n)
			}
		}
	}
	return out
}

// Positions returns costed inventory.
func (e *Engine) Positions() map[string]map[string]*Position {
	return e.Inv
}

func IsIncome(account string) bool  { return strings.HasPrefix(account, "Income:") }
func IsExpense(account string) bool { return strings.HasPrefix(account, "Expenses:") }
func IsAsset(account string) bool   { return strings.HasPrefix(account, "Assets:") }
func IsLiability(account string) bool {
	return strings.HasPrefix(account, "Liabilities:")
}
