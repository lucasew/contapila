package ledger

import (
	"testing"
	"github.com/contapila/contapila/pkg/core"
)

func TestLedgerStream(t *testing.T) {
	l, err := Load("../../testdata/fixture.beancount")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(l.Stream) != 5 {
		t.Errorf("Expected 5 directives in stream, got %d", len(l.Stream))
	}

	// Verify order
	types := []string{"*core.Transaction", "core.Note", "core.Event", "*core.Transaction", "core.Note"}
	for i, d := range l.Stream {
		actualType := ""
		switch d.(type) {
		case *core.Transaction:
			actualType = "*core.Transaction"
		case core.Note:
			actualType = "core.Note"
		case core.Event:
			actualType = "core.Event"
		}
		if actualType != types[i] {
			t.Errorf("At index %d, expected type %s, got %T", i, types[i], d)
		}
	}
}
