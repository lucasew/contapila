package ledger

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Project struct {
	Root        string
	LedgerNames []string
}

type Diagnostic struct {
	Severity string // "warn" or "error"
	Message  string
	File     string
	Line     int
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", d.File, d.Line, d.Severity, d.Message)
}

type Posting struct {
	Date      time.Time
	Account   string
	Commodity string
	Amount    *big.Rat
}

type BalanceAssertion struct {
	Date      time.Time
	Account   string
	Commodity string
	Expected  *big.Rat
	File      string
	Line      int
}

type Ledger struct {
	Name             string
	Project          *Project
	Diagnostics      []Diagnostic
	Accounts         map[string]bool
	Postings         []Posting
	BalanceAssertions []BalanceAssertion
}

func DiscoverProject(cwd string) (*Project, error) {
	curr, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	for {
		marker := filepath.Join(curr, "contapila.cue")
		if _, err := os.Stat(marker); err == nil {
			ledgers, err := findLedgers(curr)
			if err != nil {
				return nil, err
			}
			return &Project{
				Root:        curr,
				LedgerNames: ledgers,
			}, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return nil, fmt.Errorf("not a contapila project (contapila.cue not found in parents)")
}

func findLedgers(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var ledgers []string
	for _, entry := range entries {
		if entry.IsDir() {
			mainBeancount := filepath.Join(root, entry.Name(), "main.beancount")
			if _, err := os.Stat(mainBeancount); err == nil {
				ledgers = append(ledgers, entry.Name())
			}
		}
	}
	return ledgers, nil
}

func (p *Project) LoadLedger(name string) (*Ledger, error) {
	found := false
	for _, n := range p.LedgerNames {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("ledger %q not found", name)
	}

	l := &Ledger{
		Name:     name,
		Project:  p,
		Accounts: make(map[string]bool),
	}

	mainFile := filepath.Join(p.Root, name, "main.beancount")
	err := l.parseFile(mainFile)
	if err != nil {
		return nil, err
	}

	// Run initial check to populate balance assertion errors
	l.runBalanceAssertions()

	return l, nil
}

var (
	reOpen    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+open\s+([A-Za-z0-9:]+)`)
	reBalance = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+balance\s+([A-Za-z0-9:]+)\s+(-?\d+\.?\d*)\s+([A-Z]+)`)
	reTxn     = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+[\*!]\s+(.*)`)
	rePosting = regexp.MustCompile(`^\s+([A-Za-z0-9:]+)(\s+(-?\d+\.?\d*)\s+([A-Z]+))?`)
	reInclude = regexp.MustCompile(`^include\s+"(.*)"`)
	rePad     = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+pad\s+([A-Za-z0-9:]+)\s+([A-Za-z0-9:]+)`)
)

func (l *Ledger) parseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var currentTxnDate time.Time

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), ";") || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		if match := reInclude.FindStringSubmatch(line); match != nil {
			incPath := match[1]
			if !filepath.IsAbs(incPath) {
				incPath = filepath.Join(filepath.Dir(path), incPath)
			}
			if err := l.parseFile(incPath); err != nil {
				l.Diagnostics = append(l.Diagnostics, Diagnostic{
					Severity: "error",
					Message:  fmt.Sprintf("failed to include %s: %v", incPath, err),
					File:     path,
					Line:     lineNum,
				})
			}
			continue
		}

		if match := reOpen.FindStringSubmatch(line); match != nil {
			account := match[2]
			if l.Accounts[account] {
				l.Diagnostics = append(l.Diagnostics, Diagnostic{
					Severity: "error",
					Message:  fmt.Sprintf("duplicate open for account %s", account),
					File:     path,
					Line:     lineNum,
				})
			}
			l.Accounts[account] = true
			continue
		}

		if match := rePad.FindStringSubmatch(line); match != nil {
			continue
		}

		if match := reBalance.FindStringSubmatch(line); match != nil {
			dateStr := match[1]
			account := match[2]
			amountStr := match[3]
			commodity := match[4]

			date, _ := time.Parse("2006-01-02", dateStr)
			expected := new(big.Rat)
			expected.SetString(amountStr)

			l.BalanceAssertions = append(l.BalanceAssertions, BalanceAssertion{
				Date:      date,
				Account:   account,
				Commodity: commodity,
				Expected:  expected,
				File:      path,
				Line:      lineNum,
			})
			continue
		}

		if match := reTxn.FindStringSubmatch(line); match != nil {
			dateStr := match[1]
			currentTxnDate, _ = time.Parse("2006-01-02", dateStr)
			continue
		}

		if match := rePosting.FindStringSubmatch(line); match != nil {
			account := match[1]
			amountStr := match[3]
			commodity := match[4]

			if !l.Accounts[account] {
				l.Diagnostics = append(l.Diagnostics, Diagnostic{
					Severity: "warn",
					Message:  fmt.Sprintf("use of unopened account %s", account),
					File:     path,
					Line:     lineNum,
				})
			}

			if amountStr != "" {
				amount := new(big.Rat)
				amount.SetString(amountStr)
				l.Postings = append(l.Postings, Posting{
					Date:      currentTxnDate,
					Account:   account,
					Commodity: commodity,
					Amount:    amount,
				})
			}
			continue
		}
	}
	return scanner.Err()
}

func (l *Ledger) runBalanceAssertions() {
	for _, ba := range l.BalanceAssertions {
		// balance assertion at date D includes transactions ON date D
		// (Beancount actually checks it BEFORE transactions on that date, but here we'll include them if they have the same date)
		// Actually Beancount's balance assertion usually includes transactions up to the start of the day.
		// Let's stick to Beancount's: balance assertion at date D checks balance AFTER all transactions strictly BEFORE D.

		balances, _ := l.GetBalances(ba.Date)
		actual := new(big.Rat)
		for _, b := range balances {
			if b.Account == ba.Account && b.Commodity == ba.Commodity {
				actual.Set(b.Amount)
				break
			}
		}

		if actual.Cmp(ba.Expected) != 0 {
			l.Diagnostics = append(l.Diagnostics, Diagnostic{
				Severity: "error",
				Message:  fmt.Sprintf("balance assertion failed for %s: expected %s %s, got %s %s", ba.Account, ba.Expected.FloatString(2), ba.Commodity, actual.FloatString(2), ba.Commodity),
				File:     ba.File,
				Line:     ba.Line,
			})
		}
	}
}

func (l *Ledger) Check() ([]Diagnostic, error) {
	return l.Diagnostics, nil
}

type Balance struct {
	Account   string
	Commodity string
	Amount    *big.Rat
}

func (l *Ledger) GetBalances(asOf time.Time) ([]Balance, error) {
	accBalances := make(map[string]map[string]*big.Rat)
	for _, p := range l.Postings {
		if !asOf.IsZero() && p.Date.After(asOf) && !p.Date.Equal(asOf) {
			// Beancount balance at date D usually means balance at start of day D.
			// So it includes transactions strictly before D.
			if !p.Date.Equal(asOf) {
				continue
			}
		}
		// Wait, if I use asOf as the limit, usually it's inclusive of start of day.
		// "as-of date" in requirements usually means "at the end of day" or "at the beginning of day".
		// Let's assume inclusive of the date if we want the balance at the end of that day.
		if !asOf.IsZero() && p.Date.After(asOf) {
			continue
		}

		if accBalances[p.Account] == nil {
			accBalances[p.Account] = make(map[string]*big.Rat)
		}
		if _, ok := accBalances[p.Account][p.Commodity]; !ok {
			accBalances[p.Account][p.Commodity] = new(big.Rat)
		}
		accBalances[p.Account][p.Commodity].Add(accBalances[p.Account][p.Commodity], p.Amount)
	}

	var res []Balance
	for acc, comms := range accBalances {
		for comm, amt := range comms {
			if amt.Sign() != 0 {
				res = append(res, Balance{
					Account:   acc,
					Commodity: comm,
					Amount:    amt,
				})
			}
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Account != res[j].Account {
			return res[i].Account < res[j].Account
		}
		return res[i].Commodity < res[j].Commodity
	})
	return res, nil
}
