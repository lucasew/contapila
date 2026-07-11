// Package source holds Beancount (and other) text files with byte→line mapping.
package source

import (
	"os"
	"sort"
)

// File is a named UTF-8 text buffer plus a line index for diagnostics and AST Meta.
// Build once per parse; LineAt is O(log lines). Offsets are byte offsets (tree-sitter).
type File struct {
	Path string
	// Text is the file contents (UTF-8). Prefer string over []byte so callers
	// stay in the encoding domain Go and ParseString already use.
	Text string
	// starts[i] is the byte offset where 1-based line i+1 begins.
	starts []int
}

// New reads path from disk (as UTF-8 text) and builds the line index.
func New(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewString(path, string(b)), nil
}

// NewString builds a File from an already-loaded string (tests, embeds, pipes).
func NewString(path, text string) *File {
	f := &File{Path: path, Text: text}
	// ~1 entry per 40 bytes (typical ledger line); grows if denser.
	starts := make([]int, 1, len(text)/40+2)
	starts[0] = 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' && i+1 < len(text) {
			starts = append(starts, i+1)
		}
	}
	f.starts = starts
	return f
}

// LineAt returns the 1-based line containing byte offset off.
// Offsets past the end map to the last line; empty files return 1.
func (f *File) LineAt(off int) int {
	if f == nil || len(f.starts) == 0 {
		return 1
	}
	// largest i with starts[i] <= off
	i := sort.Search(len(f.starts), func(i int) bool {
		return f.starts[i] > off
	}) - 1
	if i < 0 {
		return 1
	}
	return i + 1
}

// LineAtU32 is LineAt for tree-sitter byte offsets.
func (f *File) LineAtU32(off uint32) int {
	return f.LineAt(int(off))
}

// Slice returns Text[start:end] safely (empty if out of range).
// start/end are byte offsets, matching tree-sitter.
func (f *File) Slice(start, end int) string {
	if f == nil {
		return ""
	}
	if start < 0 || end > len(f.Text) || start > end {
		return ""
	}
	return f.Text[start:end]
}
