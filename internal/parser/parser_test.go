package parser

import (
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
