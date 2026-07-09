package ledger

import (
	"strings"
	"testing"
	"time"

	"github.com/lucasew/contapila-go/pkg/core"
)

func TestJournalOutput(t *testing.T) {
	l := &Ledger{
		Stream: []core.Directive{
			&core.Transaction{Date: date("2024-01-01"), Flag: "*", Narration: "Opening"},
			core.Note{Date: date("2024-01-02"), Account: "Assets:Bank", Comment: "Test note"},
			core.Event{Date: date("2024-01-03"), Type: "type", Description: "desc"},
		},
	}

	var sb strings.Builder
	l.Journal(&sb)
	output := sb.String()

	expected := `2024-01-01 * "Opening"
2024-01-02 note Assets:Bank "Test note"
2024-01-03 event "type" "desc"
`
	if output != expected {
		t.Errorf("Journal output mismatch.\nExpected:\n%s\nGot:\n%s", expected, output)
	}
}

func date(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}
