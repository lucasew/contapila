package engine

import (
	"contapila/internal/booking"
	"contapila/internal/parser"
	"contapila/internal/project"
	"fmt"
	"os"
	"path/filepath"
)

type Ledger struct {
	Name     string
	Balances map[string]booking.Inventory
	Errors   []string
	Warnings []string
	Accounts map[string]AccountStatus
}

type AccountStatus struct {
	OpenDate  string
	CloseDate string
}

func ProcessLedger(proj *project.Project, info project.LedgerInfo) (*Ledger, error) {
	l := &Ledger{
		Name:     info.Name,
		Balances: make(map[string]booking.Inventory),
		Accounts: make(map[string]AccountStatus),
	}

	directives, err := loadDirectives(info.Path)
	if err != nil {
		return nil, err
	}

	for _, d := range directives {
		switch v := d.(type) {
		case parser.Open:
			l.Accounts[v.Account] = AccountStatus{OpenDate: v.Date}
		case parser.Close:
			if status, ok := l.Accounts[v.Account]; ok {
				status.CloseDate = v.Date
				l.Accounts[v.Account] = status
			}
		case *parser.Transaction:
			l.applyTransaction(v)
		case parser.Balance:
			l.checkBalance(v)
		}
	}

	return l, nil
}

func loadDirectives(path string) ([]parser.Directive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	directives, err := parser.Parse(f)
	if err != nil {
		return nil, err
	}

	var all []parser.Directive
	dir := filepath.Dir(path)

	for _, d := range directives {
		if inc, ok := d.(parser.Include); ok {
			incPath := inc.Path
			if !filepath.IsAbs(incPath) {
				incPath = filepath.Join(dir, incPath)
			}
			matches, err := filepath.Glob(incPath)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				sub, err := loadDirectives(m)
				if err != nil {
					return nil, err
				}
				all = append(all, sub...)
			}
		} else {
			all = append(all, d)
		}
	}
	return all, nil
}

func (l *Ledger) applyTransaction(txn *parser.Transaction) {
	for _, p := range txn.Postings {
		status, ok := l.Accounts[p.Account]
		if !ok {
			l.Warnings = append(l.Warnings, fmt.Sprintf("unopened account used: %s", p.Account))
		} else if status.CloseDate != "" && txn.Date > status.CloseDate {
			l.Errors = append(l.Errors, fmt.Sprintf("posting after close: %s on %s", p.Account, txn.Date))
		}
	}

	var postingsWithAmount []int
	var postingsWithoutAmount []int

	for i, p := range txn.Postings {
		if p.Amount != nil {
			postingsWithAmount = append(postingsWithAmount, i)
		} else {
			postingsWithoutAmount = append(postingsWithoutAmount, i)
		}
	}

	if len(postingsWithoutAmount) > 1 {
		l.Errors = append(l.Errors, fmt.Sprintf("unbalanced txn at %s: more than one residual leg", txn.Date))
		return
	}

	runningBalances := make(map[string]booking.Decimal)

	for _, i := range postingsWithAmount {
		p := &txn.Postings[i]
		inv, ok := l.Balances[p.Account]
		if !ok {
			inv = booking.NewInventory()
			l.Balances[p.Account] = inv
		}

		// Calculate balancing value BEFORE applying to inventory
		var val booking.Decimal
		var comm string

		// Priority: Price @@, then Price @, then Cost {}, then Amount
		effectiveCost := p.Cost
		if p.Price != nil {
			effectiveCost = p.Price
		}

		if effectiveCost != nil {
			val = p.Amount.Number.Mul(effectiveCost.Number)
			comm = effectiveCost.Commodity
		} else if p.Amount.Number.Sign() < 0 && inv[p.Amount.Commodity] != nil && inv[p.Amount.Commodity].Units.Sign() > 0 {
			avg := inv[p.Amount.Commodity].AverageCost()
			val = p.Amount.Number.Mul(avg)
			comm = inv[p.Amount.Commodity].CostCommodity
		} else {
			val = p.Amount.Number
			comm = p.Amount.Commodity
		}
		runningBalances[comm] = runningBalances[comm].Add(val)

		if p.Amount.Number.Sign() > 0 && effectiveCost == nil {
			l.Errors = append(l.Errors, fmt.Sprintf("%s: inventory increase requires explicit cost", txn.Date))
			continue
		}

		_, err := inv.Apply(*p.Amount, effectiveCost)
		if err != nil {
			l.Errors = append(l.Errors, fmt.Sprintf("%s: %v", txn.Date, err))
		}
	}

	if len(postingsWithoutAmount) == 1 {
		pIdx := postingsWithoutAmount[0]
		p := &txn.Postings[pIdx]

		for comm, balance := range runningBalances {
			if !balance.IsZero() {
				needed := balance.Neg()
				p.Amount = &booking.Amount{Number: needed, Commodity: comm}

				inv, ok := l.Balances[p.Account]
				if !ok {
					inv = booking.NewInventory()
					l.Balances[p.Account] = inv
				}
				inv.Apply(*p.Amount, nil)
				runningBalances[comm] = booking.NewDecimal(nil)
			}
		}
	}

	for comm, balance := range runningBalances {
		if !balance.IsZero() {
			l.Errors = append(l.Errors, fmt.Sprintf("unbalanced txn at %s: %s %s", txn.Date, balance, comm))
		}
	}
}

func (l *Ledger) checkBalance(b parser.Balance) {
	inv, ok := l.Balances[b.Account]
	if !ok {
		if !b.Amount.Number.IsZero() {
			l.Errors = append(l.Errors, fmt.Sprintf("balance assertion failed: %s has 0.00000 %s, expected %s", b.Account, b.Amount.Commodity, b.Amount.Number))
		}
		return
	}

	pos, ok := inv[b.Amount.Commodity]
	if !ok {
		if !b.Amount.Number.IsZero() {
			l.Errors = append(l.Errors, fmt.Sprintf("balance assertion failed: %s has 0.00000 %s, expected %s", b.Account, b.Amount.Commodity, b.Amount.Number))
		}
		return
	}

	if pos.Units.Cmp(b.Amount.Number) != 0 {
		l.Errors = append(l.Errors, fmt.Sprintf("balance assertion failed: %s has %s %s, expected %s", b.Account, pos.Units, b.Amount.Commodity, b.Amount.Number))
	}
}
