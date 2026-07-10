package engine

import (
	"math/big"
	"sort"
	"strings"
	"time"
)

// NetWorthTreeLine is one hierarchical net-worth row (value in op currency).
// Name is the last path segment (Fava-style); Account is the full path for links.
// Path is the tree identity for collapse (account, or account+\x1f+commodity).
type NetWorthTreeLine struct {
	Account   string
	Path      string // collapse key
	Name      string
	Depth     int
	Units     *big.Rat // nil when multi-commodity rollup / parent
	Commodity string
	Value     *big.Rat // signed, operating currency
	IsRollup  bool
	UsedCost  bool
}

// TreePathSep separates account path from commodity on multi-ccy leaf rows.
const TreePathSep = "\x1f"

// NetWorthTree returns Assets/Liabilities as a collapsible account tree.
func (l *Ledger) NetWorthTree(asOf time.Time) ([]NetWorthTreeLine, *big.Rat, error) {
	lines, total, err := l.NetWorth(asOf)
	if err != nil {
		return nil, nil, err
	}
	return netWorthTreeFromLines(lines), total, nil
}

func netWorthTreeFromLines(lines []NetWorthLine) []NetWorthTreeLine {
	if len(lines) == 0 {
		return nil
	}

	type agg struct {
		value    *big.Rat
		usedCost bool
		comms    map[string]*big.Rat // commodity -> units
		// per-commodity value for multi-ccy leaves
		commVal  map[string]*big.Rat
		commCost map[string]bool
	}
	byAcct := map[string]*agg{}
	for _, ln := range lines {
		a := byAcct[ln.Account]
		if a == nil {
			a = &agg{
				value:    big.NewRat(0, 1),
				comms:    map[string]*big.Rat{},
				commVal:  map[string]*big.Rat{},
				commCost: map[string]bool{},
			}
			byAcct[ln.Account] = a
		}
		a.value.Add(a.value, ln.Value)
		if ln.UsedCost {
			a.usedCost = true
			a.commCost[ln.Commodity] = true
		}
		if ln.Units != nil {
			if a.comms[ln.Commodity] == nil {
				a.comms[ln.Commodity] = big.NewRat(0, 1)
			}
			a.comms[ln.Commodity].Add(a.comms[ln.Commodity], ln.Units)
		}
		if a.commVal[ln.Commodity] == nil {
			a.commVal[ln.Commodity] = big.NewRat(0, 1)
		}
		a.commVal[ln.Commodity].Add(a.commVal[ln.Commodity], ln.Value)
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

	rollupVal := map[string]*big.Rat{}
	rollupCost := map[string]bool{}
	for _, n := range names {
		tot := big.NewRat(0, 1)
		cost := false
		for a, ag := range byAcct {
			if a == n || strings.HasPrefix(a, n+":") {
				tot.Add(tot, ag.value)
				if ag.usedCost {
					cost = true
				}
			}
		}
		if tot.Sign() != 0 {
			rollupVal[n] = tot
			rollupCost[n] = cost
		}
	}

	hasChild := map[string]bool{}
	for _, n := range names {
		for _, m := range names {
			if strings.HasPrefix(m, n+":") {
				hasChild[n] = true
				break
			}
		}
	}

	out := make([]NetWorthTreeLine, 0, len(names)+8)
	for _, n := range names {
		val := rollupVal[n]
		if val == nil {
			continue
		}
		row := NetWorthTreeLine{
			Account:  n,
			Path:     n,
			Name:     accountLeaf(n),
			Depth:    strings.Count(n, ":"),
			Value:    new(big.Rat).Set(val),
			IsRollup: hasChild[n],
			UsedCost: rollupCost[n],
		}
		ag := byAcct[n]
		if !hasChild[n] && ag != nil && len(ag.comms) == 1 {
			for c, u := range ag.comms {
				row.Commodity = c
				row.Units = new(big.Rat).Set(u)
			}
		}
		// Multi-commodity leaf account is a rollup of commodity sub-rows
		if !hasChild[n] && ag != nil && len(ag.comms) > 1 {
			row.IsRollup = true
		}
		out = append(out, row)

		if !hasChild[n] && ag != nil && len(ag.comms) > 1 {
			var cs []string
			for c := range ag.comms {
				cs = append(cs, c)
			}
			sort.Strings(cs)
			for _, c := range cs {
				out = append(out, NetWorthTreeLine{
					Account:   n,
					Path:      n + TreePathSep + c,
					Name:      c,
					Depth:     strings.Count(n, ":") + 1,
					Units:     new(big.Rat).Set(ag.comms[c]),
					Commodity: c,
					Value:     new(big.Rat).Set(ag.commVal[c]),
					IsRollup:  false,
					UsedCost:  ag.commCost[c],
				})
			}
		}
	}
	return out
}
