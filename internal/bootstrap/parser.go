// Package bootstrap implements a minimal, temporary Beancount parser.
// THIS IS NOT A PRODUCTION PARSER. It is used only for bootstrapping the CLI
// and running tests until the official tree-sitter grammar is available.
// See SPEC.md §5.3 and JULES.md for details.
package bootstrap

import (
	"bufio"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/lucasew/contapila-go/internal/ledger"
)

func Parse(r io.Reader) ([]ledger.Directive, error) {
	var directives []ledger.Directive
	scanner := bufio.NewScanner(r)
	var currentTxn *ledger.Transaction

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// New directive
			if currentTxn != nil {
				directives = append(directives, *currentTxn)
				currentTxn = nil
			}

			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			date, err := time.Parse("2006-01-02", parts[0])
			if err != nil {
				continue // Skip lines that don't start with a date
			}

			switch parts[1] {
			case "open":
				if len(parts) < 3 {
					continue
				}
				directives = append(directives, ledger.Open{
					Date:    date,
					Account: parts[2],
				})
			case "close":
				if len(parts) < 3 {
					continue
				}
				directives = append(directives, ledger.Close{
					Date:    date,
					Account: parts[2],
				})
			case "*", "!":
				currentTxn = &ledger.Transaction{
					Date: date,
					Flag: parts[1],
				}
				if len(parts) > 2 {
					currentTxn.Narration = strings.Join(parts[2:], " ")
				}
			}
		} else if currentTxn != nil {
			// Posting
			parts := strings.Fields(trimmed)
			if len(parts) == 0 {
				continue
			}

			p := ledger.Posting{
				Account: parts[0],
			}

			if len(parts) >= 3 {
				num, ok := new(big.Rat).SetString(parts[1])
				if ok {
					p.Amount = &ledger.Amount{
						Number:    num,
						Commodity: parts[2],
					}
				}
			}
			currentTxn.Postings = append(currentTxn.Postings, p)
		}
	}

	if currentTxn != nil {
		directives = append(directives, *currentTxn)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return directives, nil
}
