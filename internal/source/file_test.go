package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLineAt(t *testing.T) {
	f := NewString("t.beancount", "a\nbb\nccc\n")
	cases := []struct {
		off  int
		want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{4, 2},
		{5, 3},
		{8, 3},
		{100, 3},
	}
	for _, c := range cases {
		if got := f.LineAt(c.off); got != c.want {
			t.Errorf("LineAt(%d)=%d want %d", c.off, got, c.want)
		}
	}
}

func TestLineAtEmpty(t *testing.T) {
	f := NewString("empty", "")
	if f.LineAt(0) != 1 {
		t.Fatalf("empty: %d", f.LineAt(0))
	}
}

func TestSlice(t *testing.T) {
	f := NewString("t", "hello")
	if f.Slice(1, 4) != "ell" {
		t.Fatal(f.Slice(1, 4))
	}
	if f.Slice(-1, 2) != "" || f.Slice(0, 99) != "" {
		t.Fatal("expected empty on OOB")
	}
}

func TestNewReadsDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.beancount")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path != path {
		t.Fatalf("path=%s", f.Path)
	}
	if f.LineAt(6) != 2 {
		t.Fatalf("LineAt(6)=%d", f.LineAt(6))
	}
	if f.Text != "line1\nline2\n" {
		t.Fatalf("text=%q", f.Text)
	}
}
