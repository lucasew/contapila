package parser

import (
	"bufio"
	"contapila/internal/booking"
	"io"
	"regexp"
	"strings"
)

type Directive interface{}

type Transaction struct {
	Date     string
	Flag     string
	Payee    string
	Narration string
	Postings []Posting
}

type Posting struct {
	Account   string
	Amount    *booking.Amount
	Cost      *booking.Amount
	Price     *booking.Amount
}

type Option struct {
	Name  string
	Value string
}

type Include struct {
	Path string
}

type Open struct {
	Date    string
	Account string
}

type Close struct {
	Date    string
	Account string
}

type Balance struct {
	Date    string
	Account string
	Amount  booking.Amount
}

type Pad struct {
	Date     string
	Account  string
	From     string
}

type Price struct {
	Date      string
	Commodity string
	Amount    booking.Amount
}

var (
	txnRegex     = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+(?:"([^"]*)")?\s*(?:"([^"]*)")?`)
	postingRegex = regexp.MustCompile(`^\s+([A-Z][A-Za-z0-9:]+)(?:\s+(-?[\d.]+)\s+([A-Z][A-Z0-9]*))?(?:\s+\{(-?[\d.]+)\s+([A-Z][A-Z0-9]*)\})?(?:\s+@\s+(-?[\d.]+)\s+([A-Z][A-Z0-9]*))?(?:\s+@@\s+(-?[\d.]+)\s+([A-Z][A-Z0-9]*))?`)
	optionRegex  = regexp.MustCompile(`^option\s+"([^"]+)"\s+"([^"]+)"`)
	includeRegex = regexp.MustCompile(`^include\s+"([^"]+)"`)
	openRegex    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+open\s+([A-Z][A-Za-z0-9:]+)`)
	closeRegex   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+close\s+([A-Z][A-Za-z0-9:]+)`)
	balanceRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+balance\s+([A-Z][A-Za-z0-9:]+)\s+(-?[\d.]+)\s+([A-Z][A-Z0-9]*)`)
	padRegex     = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+pad\s+([A-Z][A-Za-z0-9:]+)\s+([A-Z][A-Za-z0-9:]+)`)
	priceRegex   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+price\s+([A-Z][A-Z0-9]*)\s+(-?[\d.]+)\s+([A-Z][A-Z0-9]*)`)
)

func Parse(r io.Reader) ([]Directive, error) {
	scanner := bufio.NewScanner(r)
	var directives []Directive
	var currentTxn *Transaction

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ";") || strings.TrimSpace(line) == "" {
			continue
		}

		if matches := txnRegex.FindStringSubmatch(line); matches != nil {
			currentTxn = &Transaction{
				Date:      matches[1],
				Flag:      matches[2],
				Payee:     matches[3],
				Narration: matches[4],
			}
			// If payee is empty but narration is not, shift them if only one string was provided
			// but the regex expects two.
			// Wait, the regex is `(?:"([^"]*)")?\s*(?:"([^"]*)")?`
			// If "Sell" is provided, it might match matches[3].
			if currentTxn.Payee != "" && currentTxn.Narration == "" {
				// Often narration is the first one if only one is present
				currentTxn.Narration = currentTxn.Payee
				currentTxn.Payee = ""
			}
			directives = append(directives, currentTxn)
			continue
		}

		if matches := postingRegex.FindStringSubmatch(line); matches != nil && currentTxn != nil {
			p := Posting{Account: matches[1]}
			if matches[2] != "" {
				num, _ := booking.ParseDecimal(matches[2])
				p.Amount = &booking.Amount{Number: num, Commodity: matches[3]}
			}
			if matches[4] != "" {
				num, _ := booking.ParseDecimal(matches[4])
				p.Cost = &booking.Amount{Number: num, Commodity: matches[5]}
			}
			if matches[6] != "" {
				num, _ := booking.ParseDecimal(matches[6])
				p.Price = &booking.Amount{Number: num, Commodity: matches[7]}
			} else if matches[8] != "" {
				// @@ total price
				totalNum, _ := booking.ParseDecimal(matches[8])
				if p.Amount != nil && !p.Amount.Number.IsZero() {
					unitPrice := totalNum.Quo(p.Amount.Number.Abs())
					p.Price = &booking.Amount{Number: unitPrice, Commodity: matches[9]}
				}
			}
			currentTxn.Postings = append(currentTxn.Postings, p)
			continue
		}

		currentTxn = nil

		if matches := optionRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, Option{Name: matches[1], Value: matches[2]})
		} else if matches := includeRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, Include{Path: matches[1]})
		} else if matches := openRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, Open{Date: matches[1], Account: matches[2]})
		} else if matches := closeRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, Close{Date: matches[1], Account: matches[2]})
		} else if matches := balanceRegex.FindStringSubmatch(line); matches != nil {
			num, _ := booking.ParseDecimal(matches[3])
			directives = append(directives, Balance{Date: matches[1], Account: matches[2], Amount: booking.Amount{Number: num, Commodity: matches[4]}})
		} else if matches := padRegex.FindStringSubmatch(line); matches != nil {
			directives = append(directives, Pad{Date: matches[1], Account: matches[2], From: matches[3]})
		} else if matches := priceRegex.FindStringSubmatch(line); matches != nil {
			num, _ := booking.ParseDecimal(matches[3])
			directives = append(directives, Price{Date: matches[1], Commodity: matches[2], Amount: booking.Amount{Number: num, Commodity: matches[4]}})
		}
	}

	return directives, scanner.Err()
}
