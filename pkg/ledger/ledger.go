package ledger

import (
	"os"
	"sort"

	"github.com/contapila/contapila/pkg/core"
	"github.com/contapila/contapila/pkg/parser"
)

type Ledger struct {
	Stream []core.Directive
}

func Load(filename string) (*Ledger, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	directives, err := parser.Parse(f)
	if err != nil {
		return nil, err
	}

	// Beancount files are generally chronological, but we can ensure it.
	// For this MVP, we'll keep the order as parsed (stream).

	return &Ledger{
		Stream: directives,
	}, nil
}

func (l *Ledger) Sort() {
	sort.SliceStable(l.Stream, func(i, j int) bool {
		return l.Stream[i].GetDate().Before(l.Stream[j].GetDate())
	})
}
