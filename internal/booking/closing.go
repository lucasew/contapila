package booking

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
)

// ExpandClosing turns posting metadata closing: TRUE into synthetic directives:
//
//	next day: balance Account  0 COMMODITY   (only that posting's commodity)
//	next day: close Account
//
// Rules:
//   - unit-less (residual) posting with closing → error, no inject for that posting
//   - user already wrote close for the account → warn, skip synthetic close (balance 0 still injected)
//   - multi-commodity accounts: only balance-zero the commodity on the marked posting
//
// Synthetic directives keep the txn's File/Line for diagnostics. Appended after dirs
// (booking sorts by date; same-day order is stable).
func ExpandClosing(dirs []ast.Directive) ([]ast.Directive, diag.List) {
	var diags diag.List
	userClose := map[string]bool{}
	for _, d := range dirs {
		if c, ok := d.(ast.Close); ok && c.Account != "" {
			userClose[c.Account] = true
		}
	}

	var synth []ast.Directive
	plannedClose := map[string]bool{}
	plannedBal := map[string]bool{} // account|commodity

	for _, d := range dirs {
		t, ok := d.(ast.Transaction)
		if !ok {
			continue
		}
		for _, p := range t.Postings {
			if !metaTruthy(p.Metadata, "closing") {
				continue
			}
			if p.Units == nil || p.Units.Number == nil || p.Units.Commodity == "" {
				diags.Error(t.File, t.Line, fmt.Sprintf("closing: TRUE requires units on posting %s", p.Account))
				continue
			}
			next := nextDay(t.Date)
			balKey := p.Account + "|" + p.Units.Commodity
			if !plannedBal[balKey] {
				plannedBal[balKey] = true
				synth = append(synth, ast.Balance{
					Meta:    ast.Meta{Date: next, File: t.File, Line: t.Line},
					Account: p.Account,
					Amount:  ast.Amount{Number: big.NewRat(0, 1), Commodity: p.Units.Commodity},
				})
			}
			if userClose[p.Account] {
				diags.Warn(t.File, t.Line, fmt.Sprintf("closing: TRUE but close already written for %s", p.Account))
				continue
			}
			if plannedClose[p.Account] {
				continue
			}
			plannedClose[p.Account] = true
			synth = append(synth, ast.Close{
				Meta:    ast.Meta{Date: next, File: t.File, Line: t.Line},
				Account: p.Account,
			})
		}
	}
	if len(synth) == 0 {
		return dirs, diags
	}
	out := make([]ast.Directive, 0, len(dirs)+len(synth))
	out = append(out, dirs...)
	out = append(out, synth...)
	return out, diags
}

func metaTruthy(md ast.Metadata, key string) bool {
	if len(md) == 0 {
		return false
	}
	v, ok := md[key]
	if !ok {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func nextDay(t time.Time) time.Time {
	// Calendar day in UTC (directive dates are date-only UTC).
	y, m, day := t.Date()
	return time.Date(y, m, day+1, 0, 0, 0, 0, time.UTC)
}
