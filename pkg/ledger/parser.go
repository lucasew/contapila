package ledger

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/lucasew/contapila-go/pkg/core"
)

var (
	dateRegex  = `(\d{4}-\d{2}-\d{2})`
	noteRegex  = regexp.MustCompile(`^` + dateRegex + `\s+note\s+([A-Z][A-Za-z0-9:]+)\s+"([^"]+)"`)
	eventRegex = regexp.MustCompile(`^` + dateRegex + `\s+event\s+"([^"]+)"\s+"([^"]+)"`)
	txnRegex   = regexp.MustCompile(`^` + dateRegex + `\s+([*!])\s+(?:"([^"]+)"\s+)?(?:"([^"]+)")?`)
)

// Parse is a temporary placeholder parser until the official tree-sitter grammar is available (#3).
// It only supports a subset of directives needed for MVP dogfooding.
func Parse(r io.Reader) ([]core.Directive, error) {
	var stream []core.Directive
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Note
		if m := noteRegex.FindStringSubmatch(line); m != nil {
			date, _ := time.Parse("2006-01-02", m[1])
			stream = append(stream, core.Note{
				Date:    date,
				Account: m[2],
				Comment: m[3],
			})
			continue
		}

		// Event
		if m := eventRegex.FindStringSubmatch(line); m != nil {
			date, _ := time.Parse("2006-01-02", m[1])
			stream = append(stream, core.Event{
				Date:        date,
				Type:        m[2],
				Description: m[3],
			})
			continue
		}

		// Transaction (minimal)
		if m := txnRegex.FindStringSubmatch(line); m != nil {
			date, _ := time.Parse("2006-01-02", m[1])
			payee := m[3]
			narration := m[4]
			if narration == "" && payee != "" {
				narration = payee
				payee = ""
			}
			stream = append(stream, &core.Transaction{
				Date:      date,
				Flag:      m[2],
				Payee:     payee,
				Narration: narration,
			})
			continue
		}
	}
	return stream, scanner.Err()
}

// Load loads a ledger from a file.
func Load(filename string) (*Ledger, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stream, err := Parse(f)
	if err != nil {
		return nil, err
	}

	return &Ledger{Stream: stream}, nil
}
