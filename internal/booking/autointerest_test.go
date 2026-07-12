package booking

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/internal/ast"
)

func TestParseInterestRate(t *testing.T) {
	cases := []struct {
		in      string
		ind     InterestIndicator
		alpha   string
		hasPlus bool
	}{
		{"115% CDI", IndicatorCDI, "1.15", false},
		{"115%CDI", IndicatorCDI, "1.15", false},
		{"IPCA + 10% aa", IndicatorIPCA, "1", true},
		{"IPCA+10%aa", IndicatorIPCA, "1", true},
		{"10% aa", IndicatorFixed, "1", true},
		{"10%", IndicatorFixed, "1", true},
		{"CDI", IndicatorCDI, "1", false},
	}
	for _, tc := range cases {
		ir, ok := ParseInterestRate(tc.in)
		if !ok {
			t.Fatalf("%q: parse failed", tc.in)
		}
		if ir.Indicator != tc.ind {
			t.Fatalf("%q: ind=%s want %s", tc.in, ir.Indicator, tc.ind)
		}
		wantA, _ := new(big.Rat).SetString(tc.alpha)
		if ir.Alpha.Cmp(wantA) != 0 {
			t.Fatalf("%q: alpha=%s want %s", tc.in, ir.Alpha.FloatString(4), tc.alpha)
		}
		if tc.hasPlus && (ir.PlusDaily == nil || ir.PlusDaily.Sign() == 0) {
			t.Fatalf("%q: expected plus daily", tc.in)
		}
		if !tc.hasPlus && ir.PlusDaily != nil && ir.PlusDaily.Sign() != 0 {
			t.Fatalf("%q: unexpected plus %s", tc.in, ir.PlusDaily.FloatString(8))
		}
	}
	if _, ok := ParseInterestRate("garbage"); ok {
		t.Fatal("expected fail")
	}
}

func TestIncomePassivoAccount(t *testing.T) {
	got := IncomePassivoAccount("Assets:BR:Banco:CDB:20250404")
	want := "Income:Passivo:BR:Banco:CDB:20250404"
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestExpandAutoInterestPadAndIncomeOpen(t *testing.T) {
	dirs := []ast.Directive{
		ast.Open{
			Meta:     ast.Meta{Date: d("2025-04-04"), File: "t", Line: 1},
			Account:  "Assets:BR:CDB:X",
			Currencies: []string{"BRL"},
			Metadata: ast.Metadata{"interest_rate": "115% CDI"},
		},
		ast.Open{Meta: ast.Meta{Date: d("2025-04-04")}, Account: "Assets:Cash"},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2025-04-04"), File: "t", Line: 3}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:BR:CDB:X", Units: amt("1000", "BRL")},
				{Account: "Assets:Cash", Units: amt("-1000", "BRL")},
			},
		},
		ast.Balance{
			Meta:    ast.Meta{Date: d("2025-05-04"), File: "t", Line: 10},
			Account: "Assets:BR:CDB:X",
			Amount:  ast.Amount{Number: r("1010"), Commodity: "BRL"},
		},
	}
	out, diags := ExpandAutoInterest(dirs)
	if diags.HasErrors() {
		t.Fatalf("diags=%v", diags)
	}
	var incomeOpen bool
	var pad ast.Pad
	var padOK bool
	for _, d := range out {
		switch v := d.(type) {
		case ast.Open:
			if v.Account == "Income:Passivo:BR:CDB:X" {
				incomeOpen = true
			}
		case ast.Pad:
			if v.Account == "Assets:BR:CDB:X" {
				pad = v
				padOK = true
			}
		}
	}
	if !incomeOpen {
		t.Fatal("expected income open")
	}
	if !padOK {
		t.Fatal("expected pad")
	}
	if pad.FromAccount != "Income:Passivo:BR:CDB:X" {
		t.Fatalf("from=%s", pad.FromAccount)
	}
	// day before balance
	if !pad.Date.Equal(d("2025-05-03")) {
		t.Fatalf("pad date=%s", pad.Date.Format("2006-01-02"))
	}

	// Book: pad should bring balance to 1010
	e := New()
	e.Book(out)
	if e.Diags.HasErrors() {
		t.Fatalf("book: %v", e.Diags)
	}
	got := e.balOf("Assets:BR:CDB:X", "BRL")
	if got.Cmp(r("1010")) != 0 {
		t.Fatalf("asset bal %s", got.FloatString(4))
	}
	// income absorbed -10
	inc := e.balOf("Income:Passivo:BR:CDB:X", "BRL")
	if inc.Cmp(r("-10")) != 0 {
		t.Fatalf("income bal %s want -10", inc.FloatString(4))
	}
}

func TestExpandAutoInterestNoDoublePad(t *testing.T) {
	dirs := []ast.Directive{
		ast.Open{
			Meta:     ast.Meta{Date: d("2025-01-01"), File: "t", Line: 1},
			Account:  "Assets:CDB",
			Metadata: ast.Metadata{"interest-rate": "100% CDI"},
		},
		ast.Open{Meta: ast.Meta{Date: d("2025-01-01")}, Account: "Income:Passivo:CDB"},
		ast.Pad{
			Meta:        ast.Meta{Date: d("2025-02-01")},
			Account:     "Assets:CDB",
			FromAccount: "Income:Passivo:CDB",
		},
		ast.Balance{
			Meta:    ast.Meta{Date: d("2025-02-02")},
			Account: "Assets:CDB",
			Amount:  ast.Amount{Number: r("100"), Commodity: "BRL"},
		},
	}
	out, _ := ExpandAutoInterest(dirs)
	nPad := 0
	for _, d := range out {
		if _, ok := d.(ast.Pad); ok {
			nPad++
		}
	}
	if nPad != 1 {
		t.Fatalf("pads=%d want 1", nPad)
	}
}

func TestLoadIndexDBAndProject(t *testing.T) {
	dirs := []ast.Directive{
		ast.Open{
			Meta:     ast.Meta{Date: d("2025-01-01"), File: "t", Line: 1},
			Account:  "Assets:CDB",
			Metadata: ast.Metadata{"interest_rate": "100% CDI"},
		},
		ast.Custom{
			Meta: ast.Meta{Date: d("2025-01-02")},
			Type: "index",
			Values: []ast.CustomValue{
				{Text: "CDI"},
				{Number: r("0.01")}, // 1% day for easy math
			},
		},
		ast.Transaction{
			Meta: ast.Meta{Date: d("2025-01-01")}, Flag: "*",
			Postings: []ast.Posting{
				{Account: "Assets:CDB", Units: amt("100", "BRL")},
				{Account: "Equity:Open", Units: amt("-100", "BRL")},
			},
		},
		ast.Open{Meta: ast.Meta{Date: d("2025-01-01")}, Account: "Equity:Open"},
	}
	idx := LoadIndexDB(dirs)
	if idx.IndexRate(IndicatorCDI, d("2025-01-02")).Cmp(r("0.01")) != 0 {
		t.Fatalf("idx=%s", idx.IndexRate(IndicatorCDI, d("2025-01-02")).FloatString(4))
	}
	ir, ok := ParseInterestRate("100% CDI")
	if !ok {
		t.Fatal("parse")
	}
	// open 1st: fund 100; 2nd: apply 1% → 101
	got := ProjectedUnits("Assets:CDB", ir, d("2025-01-01"), time.Time{}, dirs, idx, d("2025-01-02"))
	if got["BRL"].Cmp(r("101")) != 0 {
		t.Fatalf("projected %s", got["BRL"].FloatString(4))
	}
}

func TestInterestRateFromMetaAlias(t *testing.T) {
	_, raw, ok := InterestRateFromMeta(ast.Metadata{"interest-rate": "115% CDI"})
	if !ok || !strings.Contains(raw, "115%") {
		t.Fatalf("ok=%v raw=%q", ok, raw)
	}
}
