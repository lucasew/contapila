package parser

import (
	"testing"

	"github.com/lucasew/contapila-go/internal/ast"
)

func TestLineTableAt(t *testing.T) {
	src := []byte("a\nbb\nccc\n")
	lt := newLineTable(src)
	cases := []struct {
		off  uint32
		want int
	}{
		{0, 1},
		{1, 1},   // '\n' of line 1
		{2, 2},   // 'b'
		{4, 2},   // '\n' of line 2
		{5, 3},   // 'c'
		{8, 3},   // last '\n'
		{100, 3}, // past end → last line
	}
	for _, c := range cases {
		if got := lt.At(c.off); got != c.want {
			t.Errorf("At(%d)=%d want %d", c.off, got, c.want)
		}
	}
}

func TestLineTableEmpty(t *testing.T) {
	lt := newLineTable(nil)
	if lt.At(0) != 1 {
		t.Fatalf("empty: %d", lt.At(0))
	}
}

func TestParseDirectiveLines(t *testing.T) {
	src := []byte("2020-01-01 open Assets:Cash\n2020-01-02 open Expenses:Food\n")
	dirs, diags, err := Parse("t.beancount", src)
	if err != nil {
		t.Fatal(err)
	}
	if diags.HasErrors() {
		t.Fatalf("%v", diags)
	}
	if len(dirs) != 2 {
		t.Fatalf("dirs=%d", len(dirs))
	}
	o0, ok0 := dirs[0].(ast.Open)
	o1, ok1 := dirs[1].(ast.Open)
	if !ok0 || !ok1 {
		t.Fatalf("types %T %T", dirs[0], dirs[1])
	}
	if o0.Line != 1 || o1.Line != 2 {
		t.Fatalf("lines %d %d", o0.Line, o1.Line)
	}
}
