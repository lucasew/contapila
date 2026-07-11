package booking

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
)

func d(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.UTC)
	return t
}
func r(s string) *big.Rat {
	x, ok := new(big.Rat).SetString(s)
	if !ok {
		panic(s)
	}
	return x
}
func amt(n, c string) *ast.Amount {
	return &ast.Amount{Number: r(n), Commodity: c}
}

func TestResidualCash(t *testing.T) {
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Expenses:Food"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-02"), File: "t"},
			Flag: "*", Narration: "Lunch",
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("-30", "BRL")},
				{Account: "Expenses:Food"},
			},
		},
	})
	if e.Diags.HasErrors() {
		t.Fatalf("errors: %v", e.Diags)
	}
	got := e.balOf("Expenses:Food", "BRL")
	if got.Cmp(r("30")) != 0 {
		t.Fatalf("got %s", got.FloatString(4))
	}
}

func TestUnbalanced(t *testing.T) {
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-02"), File: "t"},
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("-30", "BRL")},
			},
		},
	})
	if !e.Diags.HasErrors() {
		t.Fatal("expected error")
	}
}

func TestOversellUsesTxnNetNotSinglePosting(t *testing.T) {
	// Same txn: sell then buy same account+commodity. Per-posting check would
	// oversell; whole-txn net is zero relative to start+buys.
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Broker:X"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-02-01"), File: "t", Line: 3}, Flag: "*",
			Postings: []ast.Posting{
				// reduce first in source order (would fail if checked alone with 0 inventory)
				{Account: "Assets:Broker:X", Units: amt("-5", "X"), Cost: &ast.CostSpec{Number: r("10"), Commodity: "BRL"}},
				{Account: "Assets:Broker:X", Units: amt("5", "X"), Cost: &ast.CostSpec{Number: r("10"), Commodity: "BRL"}},
				{Account: "Assets:Cash", Units: amt("0", "BRL")},
			},
		},
	})
	for _, d := range e.Diags {
		if strings.Contains(d.Message, "oversell") {
			t.Fatalf("false oversell on net-zero inv txn: %v", e.Diags)
		}
	}
	if e.Diags.HasErrors() {
		t.Fatalf("errors: %v", e.Diags)
	}
	pos := e.getPos("Assets:Broker:X", "X")
	if pos == nil || pos.Units.Cmp(r("0")) != 0 {
		// buy 5 sell 5 from empty → end 0
		if pos == nil || pos.Units.Sign() != 0 {
			t.Fatalf("pos=%+v", pos)
		}
	}
}

func TestOversellIsWarning(t *testing.T) {
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Broker:X"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Income:Gains"},
		// hold 5
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-10"), File: "t"}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Broker:X", Units: amt("5", "X"), Cost: &ast.CostSpec{Number: r("10"), Commodity: "BRL"}},
				{Account: "Assets:Cash"},
			},
		},
		// sell 10 → oversell
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-02-01"), File: "t", Line: 10}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Broker:X", Units: amt("-10", "X"), Price: &ast.PriceSpec{Number: r("100"), Commodity: "BRL", Total: true}},
				{Account: "Assets:Cash", Units: amt("100", "BRL")},
				{Account: "Income:Gains"},
			},
		},
	})
	if e.Diags.HasErrors() {
		t.Fatalf("oversell must not be a hard error: %v", e.Diags)
	}
	var warned bool
	for _, d := range e.Diags {
		if d.IsWarn() && strings.Contains(d.Message, "oversell") {
			warned = true
			if d.Line != 10 {
				t.Fatalf("line=%d", d.Line)
			}
		}
	}
	if !warned {
		t.Fatalf("expected oversell warn, diags=%v", e.Diags)
	}
	// inventory left intact (oversell not applied)
	pos := e.getPos("Assets:Broker:X", "X")
	if pos == nil || pos.Units.Cmp(r("5")) != 0 {
		t.Fatalf("units after oversell skip: %+v", pos)
	}
}

func TestAverageCostSell(t *testing.T) {
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Broker:X"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Income:Gains"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-10"), File: "t"}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Broker:X", Units: amt("10", "X"), Cost: &ast.CostSpec{Number: r("10"), Commodity: "BRL"}},
				{Account: "Assets:Cash"},
			},
		},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-02-10"), File: "t"}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Broker:X", Units: amt("10", "X"), Cost: &ast.CostSpec{Number: r("20"), Commodity: "BRL"}},
				{Account: "Assets:Cash"},
			},
		},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-03-10"), File: "t"}, Flag: "*", Narration: "Sell",
			Postings: []ast.Posting{
				{Account: "Assets:Broker:X", Units: amt("-5", "X"), Price: &ast.PriceSpec{Number: r("100"), Commodity: "BRL", Total: true}},
				{Account: "Assets:Cash", Units: amt("100", "BRL")},
				{Account: "Income:Gains"},
			},
		},
	})
	if e.Diags.HasErrors() {
		t.Fatalf("errors: %v", e.Diags)
	}
	pos := e.getPos("Assets:Broker:X", "X")
	if pos == nil || pos.Units.Cmp(r("15")) != 0 {
		t.Fatalf("units left: %+v", pos)
	}
	// avg should still be 15
	if pos.Avg().Cmp(r("15")) != 0 {
		t.Fatalf("avg %s", pos.Avg().FloatString(4))
	}
	// gains: proceeds 100 - cost 75 = 25, income account gets -25 (credit)
	g := e.balOf("Income:Gains", "BRL")
	if g.Cmp(r("-25")) != 0 {
		t.Fatalf("gains bal %s want -25", g.FloatString(4))
	}
}

func TestSameDayOpenAfterTxnInStreamOK(t *testing.T) {
	// Same date: txn appears before open in stream (include/file order).
	// Type rank must still open accounts before booking the txn.
	e := New()
	e.Book([]ast.Directive{
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-01"), File: "txns.beancount", Line: 10},
			Flag: "*", Narration: "before open in stream",
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("10", "BRL")},
				{Account: "Equity:Open", Units: amt("-10", "BRL")},
			},
		},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01"), File: "accounts.beancount", Line: 1}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01"), File: "accounts.beancount", Line: 2}, Account: "Equity:Open"},
	})
	for _, d := range e.Diags {
		if strings.Contains(d.Message, "not opened") || strings.Contains(d.Message, "before open") {
			t.Fatalf("unexpected: %v", d)
		}
	}
	if _, ok := e.Open["Assets:Cash"]; !ok {
		t.Fatal("open missing")
	}
}

func TestSameDayOpenBeforeTxnOK(t *testing.T) {
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-01-01")}, Account: "Equity:Open"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-01"), File: "t", Line: 1},
			Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("10", "BRL")},
				{Account: "Equity:Open", Units: amt("-10", "BRL")},
			},
		},
	})
	for _, d := range e.Diags {
		if strings.Contains(d.Message, "not opened") || strings.Contains(d.Message, "before open") {
			t.Fatalf("unexpected: %v", d)
		}
	}
}

func TestTxnBeforeOpenDateNotOpened(t *testing.T) {
	// Date sort books the earlier txn before the later open is registered,
	// so this surfaces as "not opened" (not "before open").
	e := New()
	e.Book([]ast.Directive{
		ast.Open{Meta: ast.Meta{Date: d("2020-02-01")}, Account: "Assets:Cash"},
		ast.Open{Meta: ast.Meta{Date: d("2020-02-01")}, Account: "Equity:Open"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2020-01-15"), File: "t", Line: 1},
			Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:Cash", Units: amt("10", "BRL")},
				{Account: "Equity:Open", Units: amt("-10", "BRL")},
			},
		},
	})
	var notOpened bool
	for _, d := range e.Diags {
		if strings.Contains(d.Message, "not opened") {
			notOpened = true
		}
	}
	if !notOpened {
		t.Fatalf("expected not opened for txn before open date, diags=%v", e.Diags)
	}
}
