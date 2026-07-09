package parser

import (
	"bufio"
	"github.com/lucasew/contapila-go/internal/ledger"
	"io"
	"math/big"
	"strings"
	"time"
)

func Parse(r io.Reader) ([]ledger.Directive, error) {
	var directives []ledger.Directive
	scanner := bufio.NewScanner(r)
	var currentTxn *ledger.Transaction

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
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
			case "balance":
				if len(parts) < 5 {
					continue
				}
				num, ok := new(big.Rat).SetString(parts[3])
				if !ok {
					continue
				}
				directives = append(directives, ledger.Balance{
					Date:    date,
					Account: parts[2],
					Amount: ledger.Amount{
						Number:    num,
						Commodity: parts[4],
					},
				})
			case "pad":
				if len(parts) < 4 {
					continue
				}
				directives = append(directives, ledger.Pad{
					Date:          date,
					Account:       parts[2],
					SourceAccount: parts[3],
				})
			case "*", "!":
				currentTxn = &ledger.Transaction{
					Date: date,
					Flag: parts[1],
				}
				if len(parts) > 2 {
					// Very simple payee/narration parsing
					rem := strings.TrimSpace(line[strings.Index(line, parts[1])+1:])
					currentTxn.Narration = rem
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
