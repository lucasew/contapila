package parser

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

type Directive interface {
	isDirective()
}

type Include struct {
	Path string
}

func (i Include) isDirective() {}

type Option struct {
	Name  string
	Value string
}

func (o Option) isDirective() {}

type Commodity struct {
	Name string
}

func (c Commodity) isDirective() {}

type Open struct {
	Account    string
	Currencies []string
}

func (o Open) isDirective() {}

type Close struct {
	Account string
}

func (c Close) isDirective() {}

type Transaction struct {
	Raw      string
	Postings []string
}

func (t Transaction) isDirective() {}

var (
	includeRegex   = regexp.MustCompile(`^include\s+"([^"]+)"`)
	optionRegex    = regexp.MustCompile(`^option\s+"([^"]+)"\s+"([^"]+)"`)
	commodityRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+commodity\s+([A-Z0-9._/'-]+)`)
	openRegex      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+open\s+([A-Za-z0-9:-]+)(?:\s+([A-Z0-9._/'-]+(?:,[A-Z0-9._/'-]+)*))?`)
	closeRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+close\s+([A-Za-z0-9:-]+)`)
	txnStartRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+[*!]`)
)

func ParseFile(path string) ([]Directive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func Parse(r io.Reader) ([]Directive, error) {
	var directives []Directive
	scanner := bufio.NewScanner(r)
	var currentTxn *Transaction

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if currentTxn != nil && strings.HasPrefix(line, "  ") {
			currentTxn.Postings = append(currentTxn.Postings, line)
			continue
		}
		currentTxn = nil

		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if m := includeRegex.FindStringSubmatch(line); m != nil {
			directives = append(directives, Include{Path: m[1]})
			continue
		}
		if m := optionRegex.FindStringSubmatch(line); m != nil {
			directives = append(directives, Option{Name: m[1], Value: m[2]})
			continue
		}
		if m := commodityRegex.FindStringSubmatch(line); m != nil {
			directives = append(directives, Commodity{Name: m[1]})
			continue
		}
		if m := openRegex.FindStringSubmatch(line); m != nil {
			var currencies []string
			if m[2] != "" {
				currencies = strings.Split(m[2], ",")
			}
			directives = append(directives, Open{Account: m[1], Currencies: currencies})
			continue
		}
		if m := closeRegex.FindStringSubmatch(line); m != nil {
			directives = append(directives, Close{Account: m[1]})
			continue
		}
		if txnStartRegex.MatchString(line) {
			txn := &Transaction{Raw: line}
			directives = append(directives, txn)
			currentTxn = txn
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return directives, nil
}
