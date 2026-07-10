package engine

import (
	"math/big"
	"sort"
	"strings"
	"time"
)

// BalanceTreeLine is one hierarchical balances row (native units per commodity).
// Name is the last path segment; Account is the full path for links.
// Path is the tree identity for collapse (account, or account+\x1f+commodity).
type BalanceTreeLine struct {
	Account   string
	Path      string
	Name      string
	Depth     int
	Amount    *big.Rat // nil for pure structural / multi-ccy parents
	Commodity string
	IsRollup  bool
}

// BalancesTree returns all non-zero balances as a Fava-style account tree.
func (l *Ledger) BalancesTree(asOf time.Time) []BalanceTreeLine {
	return balancesTreeFromMap(l.BalancesAsOf(asOf))
}

func balancesTreeFromMap(bals map[string]map[string]*big.Rat) []BalanceTreeLine {
	// account -> commodity -> units (skip zeros)
	byAcct := map[string]map[string]*big.Rat{}
	for acct, byComm := range bals {
		for comm, n := range byComm {
			if n == nil || n.Sign() == 0 {
				continue
			}
			if byAcct[acct] == nil {
				byAcct[acct] = map[string]*big.Rat{}
			}
			byAcct[acct][comm] = new(big.Rat).Set(n)
		}
	}
	if len(byAcct) == 0 {
		return nil
	}

	nodes := map[string]bool{}
	for a := range byAcct {
		parts := strings.Split(a, ":")
		for i := 1; i <= len(parts); i++ {
			nodes[strings.Join(parts[:i], ":")] = true
		}
	}
	var names []string
	for n := range nodes {
		names = append(names, n)
	}
	sort.Strings(names)

	hasChild := map[string]bool{}
	for _, n := range names {
		for _, m := range names {
			if strings.HasPrefix(m, n+":") {
				hasChild[n] = true
				break
			}
		}
	}

	// A prefix node is "present" if any leaf account is under it.
	hasBalanceUnder := map[string]bool{}
	for _, n := range names {
		for a := range byAcct {
			if a == n || strings.HasPrefix(a, n+":") {
				hasBalanceUnder[n] = true
				break
			}
		}
	}

	out := make([]BalanceTreeLine, 0, len(names)+8)
	for _, n := range names {
		if !hasBalanceUnder[n] {
			continue
		}
		ag := byAcct[n]
		row := BalanceTreeLine{
			Account:  n,
			Path:     n,
			Name:     accountLeaf(n),
			Depth:    strings.Count(n, ":"),
			IsRollup: hasChild[n],
		}
		if !hasChild[n] && ag != nil && len(ag) == 1 {
			for c, u := range ag {
				row.Commodity = c
				row.Amount = new(big.Rat).Set(u)
			}
		}
		if !hasChild[n] && ag != nil && len(ag) > 1 {
			row.IsRollup = true
		}
		// Structural parents (no own balances) are always rollups
		if hasChild[n] {
			row.IsRollup = true
		}
		out = append(out, row)

		if !hasChild[n] && ag != nil && len(ag) > 1 {
			var cs []string
			for c := range ag {
				cs = append(cs, c)
			}
			sort.Strings(cs)
			for _, c := range cs {
				out = append(out, BalanceTreeLine{
					Account:   n,
					Path:      n + TreePathSep + c,
					Name:      c,
					Depth:     strings.Count(n, ":") + 1,
					Amount:    new(big.Rat).Set(ag[c]),
					Commodity: c,
					IsRollup:  false,
				})
			}
		}
	}
	return out
}
