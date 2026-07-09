package parser

import (
	"bufio"
	"contapila/internal/model"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var (
	optionRegex      = regexp.MustCompile(`^option\s+"([^"]+)"\s+"([^"]+)"`)
	includeRegex     = regexp.MustCompile(`^include\s+"([^"]+)"`)
	priceRegex       = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+price\s+([A-Z][A-Z0-9._\-\']{0,23})\s+([\-\+]?[0-9]*\.?[0-9]+)\s+([A-Z][A-Z0-9._\-\']{0,23})`)
	transactionRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([\*\!])(?:\s+"([^"]*)")?(?:\s+"([^"]*)")?`)
	postingRegex     = regexp.MustCompile(`^\s+([A-Z][A-Za-z0-9\:]+)(?:\s+([\-\+]?[0-9]*\.?[0-9]+)\s+([A-Z][A-Z0-9._\-\']{0,23}))?(?:\s+\{([^\}]+)\})?(?:\s+@\s+([\-\+]?[0-9]*\.?[0-9]+\s+[A-Z][A-Z0-9._\-\']{0,23}))?`)
)

func Parse(r io.Reader) ([]model.Directive, error) {
	var directives []model.Directive
	scanner := bufio.NewScanner(r)
	var currentTxn *model.Transaction

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// fmt.Printf("Line: %q\n", line)
		// Handle transactions and postings
		if matches := transactionRegex.FindStringSubmatch(line); matches != nil {
			date, _ := time.Parse("2006-01-02", matches[1])
			payee := matches[3]
			narration := matches[4]
			if narration == "" {
				narration = payee
				payee = ""
			}
			currentTxn = &model.Transaction{
				Date:      date,
				Flag:      matches[2],
				Payee:     payee,
				Narration: narration,
			}
			directives = append(directives, currentTxn)
			continue
		}

		if currentTxn != nil && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			if matches := postingRegex.FindStringSubmatch(line); matches != nil {
				p := model.Posting{
					Account: matches[1],
				}
				if matches[2] != "" {
					val := new(big.Rat)
					val.SetString(matches[2])
					p.Units = model.Amount{
						Value:    val,
						Currency: matches[3],
					}
				}
				if matches[4] != "" {
					// Simplified cost parsing: assume "123.45 CUR"
					parts := strings.Fields(matches[4])
					if len(parts) >= 2 {
						val := new(big.Rat)
						val.SetString(parts[0])
						p.Cost = &model.Amount{
							Value:    val,
							Currency: parts[1],
						}
					}
				}
				// Price @ ... is similar if needed, but for networth we mostly need Units + CostBasis
				currentTxn.Postings = append(currentTxn.Postings, p)
				continue
			}
		}

		// Other directives
		currentTxn = nil

		if matches := priceRegex.FindStringSubmatch(line); matches != nil {
			date, _ := time.Parse("2006-01-02", matches[1])
			val := new(big.Rat)
			val.SetString(matches[3])
			directives = append(directives, &model.PriceDirective{
				Price: model.Price{
					Date:      date,
					Commodity: matches[2],
					Value:     val,
					Target:    matches[4],
				},
			})
		} else if matches := optionRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, &model.Option{
				Name:  matches[1],
				Value: matches[2],
			})
		} else if matches := includeRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, &model.Include{
				Path: matches[1],
			})
		}
	}

	return directives, scanner.Err()
}

func parseAmount(s string) (model.Amount, error) {
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return model.Amount{}, fmt.Errorf("invalid amount: %s", s)
	}
	val := new(big.Rat)
	if _, ok := val.SetString(parts[0]); !ok {
		return model.Amount{}, fmt.Errorf("invalid number: %s", parts[0])
	}
	return model.Amount{Value: val, Currency: parts[1]}, nil
}
