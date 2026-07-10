package booking

import (
	"strings"
	"testing"

	"github.com/lucasew/contapila-go/internal/ast"
	"github.com/lucasew/contapila-go/internal/diag"
)

func TestExpandClosingHappyPath(t *testing.T) {
	dirs := []ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2024-01-01")}, Account: "Assets:Pending:Refund"},
		ast.Open{Meta: ast.Meta{Date: d("2024-01-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2024-01-01")}, Account: "Equity:Open"},
		// fund pending
		ast.Transaction{
			Meta: ast.Meta{Date: d("2024-01-05"), File: "j.beancount", Line: 5},
			Flag: "*", Narration: "fund",
			Postings: []ast.Posting{
				{Account: "Assets:Pending:Refund", Units: amt("50", "BRL")},
				{Account: "Equity:Open", Units: amt("-50", "BRL")},
			},
		},
		// drain to zero + mark close
		ast.Transaction{
			Meta:      ast.Meta{Date: d("2024-01-10"), File: "j.beancount", Line: 12},
			Flag:      "*",
			Narration: "drain",
			Postings: []ast.Posting{
				{
					Account:  "Assets:Pending:Refund",
					Units:    amt("-50", "BRL"),
					Metadata: ast.Metadata{"closing": "TRUE"},
				},
				{Account: "Assets:Cash", Units: amt("50", "BRL")},
			},
		},
	}
	out, diags := ExpandClosing(dirs)
	if diags.HasErrors() {
		t.Fatalf("diags: %v", diags)
	}
	if len(out) != len(dirs)+2 {
		t.Fatalf("want +2 synth, got %d dirs (was %d)", len(out), len(dirs))
	}
	bal, ok := out[len(dirs)].(ast.Balance)
	if !ok || bal.Account != "Assets:Pending:Refund" || bal.Amount.Commodity != "BRL" || bal.Amount.Number.Sign() != 0 {
		t.Fatalf("balance=%+v", out[len(dirs)])
	}
	if !bal.Date.Equal(d("2024-01-11")) {
		t.Fatalf("balance date=%s", bal.Date.Format("2006-01-02"))
	}
	cl, ok := out[len(dirs)+1].(ast.Close)
	if !ok || cl.Account != "Assets:Pending:Refund" || !cl.Date.Equal(d("2024-01-11")) {
		t.Fatalf("close=%+v", out[len(dirs)+1])
	}
	// End-to-end: book expanded stream; balance 0 holds after drain.
	e := New()
	e.Book(out)
	if e.Diags.HasErrors() {
		t.Fatalf("book: %v", e.Diags)
	}
	if _, ok := e.Close["Assets:Pending:Refund"]; !ok {
		t.Fatal("expected close recorded")
	}
}

func TestExpandClosingUnitlessError(t *testing.T) {
	dirs := []ast.Directive{
		ast.Transaction{
			Meta: ast.Meta{Date: d("2024-01-10"), File: "j", Line: 3},
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("-10", "BRL")},
				{Account: "Expenses:X", Metadata: ast.Metadata{"closing": "TRUE"}},
			},
		},
	}
	out, diags := ExpandClosing(dirs)
	if !diags.HasErrors() {
		t.Fatal("expected error")
	}
	if len(out) != len(dirs) {
		t.Fatalf("should not inject on error, got %d", len(out))
	}
	if !hasMsg(diags, "unit") && !hasMsg(diags, "units") {
		t.Fatalf("msg=%v", diags)
	}
}

func TestExpandClosingExistingCloseWarn(t *testing.T) {
	dirs := []ast.Directive{
		ast.Close{Meta: ast.Meta{Date: d("2024-01-20")}, Account: "Assets:Pending:Refund"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2024-01-10"), File: "j", Line: 5},
			Postings: []ast.Posting{
				{
					Account:  "Assets:Pending:Refund",
					Units:    amt("-1", "BRL"),
					Metadata: ast.Metadata{"closing": "true"},
				},
				{Account: "Assets:Cash", Units: amt("1", "BRL")},
			},
		},
	}
	out, diags := ExpandClosing(dirs)
	if diags.HasErrors() {
		t.Fatalf("errors: %v", diags)
	}
	if !hasSeverity(diags, diag.Warn) {
		t.Fatalf("expected warn, got %v", diags)
	}
	// balance 0 still injected; synthetic close skipped
	var totalClose, totalBal int
	for _, d := range out {
		switch d.(type) {
		case ast.Close:
			totalClose++
		case ast.Balance:
			totalBal++
		}
	}
	if totalClose != 1 {
		t.Fatalf("closes=%d want only user close", totalClose)
	}
	if totalBal != 1 {
		t.Fatalf("balances=%d want 1 synth", totalBal)
	}
}

func TestExpandClosingOnlyMarkedCommodity(t *testing.T) {
	dirs := []ast.Directive{
		ast.Transaction{
			Meta: ast.Meta{Date: d("2024-02-01"), File: "j", Line: 1},
			Postings: []ast.Posting{
				{
					Account:  "Assets:Broker",
					Units:    amt("-10", "USD"),
					Metadata: ast.Metadata{"closing": "TRUE"},
				},
				{Account: "Assets:Cash", Units: amt("50", "BRL")},
			},
		},
	}
	out, diags := ExpandClosing(dirs)
	if diags.HasErrors() {
		t.Fatalf("%v", diags)
	}
	var bals []ast.Balance
	for _, d := range out {
		if b, ok := d.(ast.Balance); ok {
			bals = append(bals, b)
		}
	}
	if len(bals) != 1 || bals[0].Amount.Commodity != "USD" {
		t.Fatalf("bals=%+v", bals)
	}
}

func hasMsg(list diag.List, sub string) bool {
	sub = strings.ToLower(sub)
	for _, d := range list {
		if strings.Contains(strings.ToLower(d.Message), sub) {
			return true
		}
	}
	return false
}

func hasSeverity(list diag.List, s diag.Severity) bool {
	for _, d := range list {
		if d.Severity == s {
			return true
		}
	}
	return false
}
