package parser

import (
	"strings"
	"testing"
	"github.com/contapila/contapila/pkg/core"
)

func TestParseNotesAndEvents(t *testing.T) {
	input := `
2024-01-01 * "Opening"
  Assets:Bank  100.00 USD
  Equity:Opening

2024-01-02 note Assets:Bank "Test note"
2024-01-03 event "type" "desc"
`
	directives, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(directives) != 3 {
		t.Errorf("Expected 3 directives, got %d", len(directives))
	}

	note, ok := directives[1].(core.Note)
	if !ok {
		t.Fatalf("Expected Note, got %T", directives[1])
	}
	if note.Account != "Assets:Bank" || note.Comment != "Test note" {
		t.Errorf("Note mismatch: %+v", note)
	}

	event, ok := directives[2].(core.Event)
	if !ok {
		t.Fatalf("Expected Event, got %T", directives[2])
	}
	if event.Type != "type" || event.Description != "desc" {
		t.Errorf("Event mismatch: %+v", event)
	}
}
