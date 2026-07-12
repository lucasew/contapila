package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLineAt(t *testing.T) {
	// "a\nbb\nccc\n" line starts: 0, 2, 5
	multi := NewString("t.beancount", "a\nbb\nccc\n")
	// no trailing newline: "a\nb" line starts: 0, 2
	noNL := NewString("no-nl", "a\nb")
	single := NewString("one", "hello")
	empty := NewString("empty", "")

	tests := []struct {
		name string
		f    *File
		off  int
		want int
	}{
		// multi-line: first / last byte per line, mid-line, newline byte, past EOF
		{name: "multi first byte", f: multi, off: 0, want: 1},
		{name: "multi newline of line1", f: multi, off: 1, want: 1},
		{name: "multi first of line2", f: multi, off: 2, want: 2},
		{name: "multi mid line2", f: multi, off: 3, want: 2},
		{name: "multi last of line2", f: multi, off: 4, want: 2},
		{name: "multi first of line3", f: multi, off: 5, want: 3},
		{name: "multi mid line3", f: multi, off: 6, want: 3},
		{name: "multi last content byte", f: multi, off: 7, want: 3},
		{name: "multi final newline", f: multi, off: 8, want: 3},
		{name: "multi past EOF", f: multi, off: 100, want: 3},
		{name: "multi one past end", f: multi, off: 9, want: 3},
		{name: "multi negative off", f: multi, off: -1, want: 1},

		// no trailing newline
		{name: "noNL first", f: noNL, off: 0, want: 1},
		{name: "noNL newline", f: noNL, off: 1, want: 1},
		{name: "noNL last byte", f: noNL, off: 2, want: 2},
		{name: "noNL past EOF", f: noNL, off: 3, want: 2},

		// single line
		{name: "single first", f: single, off: 0, want: 1},
		{name: "single mid", f: single, off: 2, want: 1},
		{name: "single last", f: single, off: 4, want: 1},
		{name: "single past EOF", f: single, off: 5, want: 1},

		// empty file: always line 1
		{name: "empty zero", f: empty, off: 0, want: 1},
		{name: "empty past EOF", f: empty, off: 1, want: 1},
		{name: "empty far past", f: empty, off: 100, want: 1},

		// nil receiver
		{name: "nil receiver", f: nil, off: 0, want: 1},
		{name: "nil past", f: nil, off: 99, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.f.LineAt(tt.off); got != tt.want {
				t.Errorf("LineAt(%d) = %d; want %d", tt.off, got, tt.want)
			}
		})
	}
}

func TestLineAtU32(t *testing.T) {
	f := NewString("t", "a\nb\n")
	if got := f.LineAtU32(2); got != 2 {
		t.Errorf("LineAtU32(2) = %d; want 2", got)
	}
	if got := f.LineAtU32(100); got != 2 {
		t.Errorf("LineAtU32(100) = %d; want 2", got)
	}
}

func TestSlice(t *testing.T) {
	f := NewString("t", "hello")

	tests := []struct {
		name       string
		f          *File
		start, end int
		want       string
	}{
		{name: "middle", f: f, start: 1, end: 4, want: "ell"},
		{name: "full", f: f, start: 0, end: 5, want: "hello"},
		{name: "first byte", f: f, start: 0, end: 1, want: "h"},
		{name: "last byte", f: f, start: 4, end: 5, want: "o"},
		{name: "empty range in bounds", f: f, start: 2, end: 2, want: ""},
		{name: "empty at end", f: f, start: 5, end: 5, want: ""},
		{name: "negative start", f: f, start: -1, end: 2, want: ""},
		{name: "end past len", f: f, start: 0, end: 99, want: ""},
		{name: "start past end", f: f, start: 3, end: 1, want: ""},
		{name: "start past len", f: f, start: 6, end: 7, want: ""},
		{name: "both negative", f: f, start: -2, end: -1, want: ""},
		{name: "nil receiver", f: nil, start: 0, end: 1, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.f.Slice(tt.start, tt.end); got != tt.want {
				t.Errorf("Slice(%d, %d) = %q; want %q", tt.start, tt.end, got, tt.want)
			}
		})
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

func TestNewMissingFile(t *testing.T) {
	_, err := New(filepath.Join(t.TempDir(), "missing.beancount"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
