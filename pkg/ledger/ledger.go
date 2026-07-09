package ledger

import (
	"fmt"
	"io"

	"github.com/lucasew/contapila-go/pkg/core"
)

// Ledger represents a loaded Beancount ledger.
type Ledger struct {
	Stream []core.Directive
}

// Journal prints the ledger stream to the given writer.
func (l *Ledger) Journal(w io.Writer) {
	for _, d := range l.Stream {
		dateStr := d.GetDate().Format("2006-01-02")
		switch v := d.(type) {
		case *core.Transaction:
			fmt.Fprintf(w, "%s %s", dateStr, v.Flag)
			if v.Payee != "" {
				fmt.Fprintf(w, " \"%s\"", v.Payee)
			}
			fmt.Fprintf(w, " \"%s\"\n", v.Narration)
		case core.Note:
			fmt.Fprintf(w, "%s note %s \"%s\"\n", dateStr, v.Account, v.Comment)
		case core.Event:
			fmt.Fprintf(w, "%s event \"%s\" \"%s\"\n", dateStr, v.Type, v.Description)
		}
	}
}
