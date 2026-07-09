package booking

import (
	"math/big"
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
