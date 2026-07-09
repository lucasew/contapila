package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/contapila/contapila/pkg/core"
)

var (
	dateRegex   = `(\d{4}-\d{2}-\d{2})`
	noteRegex   = regexp.MustCompile(`^` + dateRegex + `\s+note\s+([A-Z][A-Za-z0-9:]+)\s+"([^"]+)"`)
	eventRegex  = regexp.MustCompile(`^` + dateRegex + `\s+event\s+"([^"]+)"\s+"([^"]+)"`)
	txnRegex    = regexp.MustCompile(`^` + dateRegex + `\s+([*!])\s+(?:"([^"]+)"\s+)?(?:"([^"]+)")?`)
	postingRegex = regexp.MustCompile(`^\s+([A-Z][A-Za-z0-9:]+)(?:\s+(-?\d+\.\d+)\s+([A-Z]+))?`)
)

func Parse(r io.Reader) ([]core.Directive, error) {
	var directives []core.Directive
	scanner := bufio.NewScanner(r)
	var currentTxn *core.Transaction

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" || strings.HasPrefix(trimmedLine, ";") || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Handle Note
		if matches := noteRegex.FindStringSubmatch(line); matches != nil {
			date, _ := time.Parse("2006-01-02", matches[1])
			directives = append(directives, core.Note{
				Date:    date,
				Account: matches[2],
				Comment: matches[3],
			})
			currentTxn = nil
			continue
		}

		// Handle Event
		if matches := eventRegex.FindStringSubmatch(line); matches != nil {
			date, _ := time.Parse("2006-01-02", matches[1])
			directives = append(directives, core.Event{
				Date:        date,
				Type:        matches[2],
				Description: matches[3],
			})
			currentTxn = nil
			continue
		}

		// Handle Transaction
		if matches := txnRegex.FindStringSubmatch(line); matches != nil {
			date, _ := time.Parse("2006-01-02", matches[1])
			payee := matches[3]
			narration := matches[4]
			if narration == "" && payee != "" {
				narration = payee
				payee = ""
			}
			currentTxn = &core.Transaction{
				Date:      date,
				Flag:      matches[2],
				Payee:     payee,
				Narration: narration,
			}
			directives = append(directives, currentTxn)
			continue
		}

		// Handle Postings
		if currentTxn != nil {
			if matches := postingRegex.FindStringSubmatch(line); matches != nil {
				currentTxn.Postings = append(currentTxn.Postings, core.Posting{
					Account:  matches[1],
					Amount:   matches[2],
					Currency: matches[3],
				})
				continue
			}
		}

		currentTxn = nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning error: %w", err)
	}

	return directives, nil
}
