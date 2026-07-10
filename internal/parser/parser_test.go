package parser

import (
	"strings"
	"testing"

	"github.com/lucasew/contapila-go/internal/ast"
)

func TestParseBasic(t *testing.T) {
	src := []byte(`
2020-01-01 open Assets:Cash
2020-01-01 open Expenses:Food
2020-01-02 * "Lunch"
  Assets:Cash  -30.00 BRL
  Expenses:Food
`)
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags.HasErrors() {
		t.Fatalf("diags: %v", diags)
	}
	var opens, txns int
	for _, d := range dirs {
		switch d.(type) {
		case ast.Open:
			opens++
		case ast.Transaction:
			txns++
		}
	}
	if opens != 2 || txns != 1 {
		t.Fatalf("opens=%d txns=%d dirs=%d", opens, txns, len(dirs))
	}
}

func TestParseOpenCurrencyAndMetaWarn(t *testing.T) {
	src := []byte(`
2020-01-01 open Assets:Cash BRL
  institution: "Banco"
2020-01-02 * "Lunch" #todo
  role: "meal"
  Assets:Cash  -30.00BRL
  Expenses:Food
2020-01-03 query "q" "select *"
2020-01-04 custom "fava-option" "True"
2020-01-05 document Assets:Cash "personal/docs/by-account/Assets/Cash/20200105_x.txt"
`)
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	var open ast.Open
	var txn ast.Transaction
	var doc ast.Document
	for _, d := range dirs {
		switch v := d.(type) {
		case ast.Open:
			open = v
		case ast.Transaction:
			txn = v
		case ast.Document:
			doc = v
		}
	}
	if open.Account != "Assets:Cash" || len(open.Currencies) != 1 || open.Currencies[0] != "BRL" {
		t.Fatalf("open=%+v", open)
	}
	if len(txn.Tags) != 1 || txn.Tags[0] != "todo" {
		t.Fatalf("tags=%v", txn.Tags)
	}
	if txn.Postings[0].Units == nil || txn.Postings[0].Units.Commodity != "BRL" {
		t.Fatalf("posting units=%+v", txn.Postings[0].Units)
	}
	if doc.Account != "Assets:Cash" || doc.Path != "personal/docs/by-account/Assets/Cash/20200105_x.txt" {
		t.Fatalf("document=%+v", doc)
	}
	// metadata + query/custom should warn; document should parse (not warn skip)
	var hasMeta, hasQuery, hasCustom bool
	for _, d := range diags {
		if strings.Contains(d.Message, "metadata") {
			hasMeta = true
		}
		if strings.Contains(d.Message, "query") {
			hasQuery = true
		}
		if strings.Contains(d.Message, "custom") {
			hasCustom = true
		}
		if strings.Contains(d.Message, "document") {
			t.Fatalf("document should not warn-skip: %v", d.Message)
		}
	}
	if !hasMeta || !hasQuery || !hasCustom {
		t.Fatalf("expected meta/query/custom warns, diags=%v", diags)
	}
}

func TestParsePayeeAndNarration(t *testing.T) {
	src := []byte(`
2020-01-01 open Assets:Cash
2020-01-01 open Expenses:Food
2020-01-05 * "Restaurant Foo" "Lunch"
  Assets:Cash  -30.00 BRL
  Expenses:Food
2020-01-06 * "Coffee only narration"
  Assets:Cash  -10.00 BRL
  Expenses:Food
`)
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags.HasErrors() {
		t.Fatalf("diags: %v", diags)
	}
	var both, narrOnly *ast.Transaction
	for _, d := range dirs {
		txn, ok := d.(ast.Transaction)
		if !ok {
			continue
		}
		switch txn.Date.Format("2006-01-02") {
		case "2020-01-05":
			t := txn
			both = &t
		case "2020-01-06":
			t := txn
			narrOnly = &t
		}
	}
	if both == nil || narrOnly == nil {
		t.Fatalf("missing txns both=%v narrOnly=%v", both, narrOnly)
	}
	if both.Payee != "Restaurant Foo" || both.Narration != "Lunch" {
		t.Fatalf("payee+narration: payee=%q narration=%q", both.Payee, both.Narration)
	}
	if narrOnly.Payee != "" || narrOnly.Narration != "Coffee only narration" {
		t.Fatalf("narration-only: payee=%q narration=%q", narrOnly.Payee, narrOnly.Narration)
	}
}
