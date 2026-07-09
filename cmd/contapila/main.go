package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/lucasew/contapila-go/internal/diag"
	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/parser"
	"github.com/lucasew/contapila-go/internal/period"
	"github.com/lucasew/contapila-go/internal/web"
	"github.com/lucasew/contapila-go/pkg/project"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:           "contapila",
		Short:         "Contapila — Beancount-class ledger in Go",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(statusCmd(), checkCmd(), balancesCmd(), journalCmd(), pnlCmd(), networthCmd(), accountCmd(), parseCmd(), webCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return cwd
}

func printDiags(ds diag.List) {
	for _, d := range ds {
		fmt.Fprintln(os.Stderr, d.String())
	}
}



func withLedgers(args []string, fn func(*engine.Ledger) error) error {
	p, pdb, pdiags, err := engine.OpenProject(mustCwd())
	if err != nil {
		return err
	}
	printDiags(pdiags)
	names := args
	if len(names) == 0 {
		names = engine.LedgerNames(p)
		if len(names) == 0 {
			return fmt.Errorf("zero ledgers found")
		}
	}
	var failed bool
	for _, name := range names {
		l, err := engine.OpenLedger(p, pdb, name)
		if err != nil {
			return err
		}
		if err := fn(l); err != nil {
			failed = true
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if failed {
		return fmt.Errorf("one or more ledgers failed")
	}
	return nil
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use: "status", Aliases: []string{"doctor"}, Short: "Show project status",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := project.OpenProject(mustCwd())
			if err != nil {
				return err
			}
			fmt.Printf("Project root:      %s\n", p.Root)
			fmt.Printf("contapila.cue:     %s\n", filepath.Join(p.Root, "contapila.cue"))
			if len(p.Ledgers) == 0 {
				return fmt.Errorf("zero ledgers found")
			}
			fmt.Printf("Ledgers (%d):\n", len(p.Ledgers))
			for _, l := range p.Ledgers {
				fmt.Printf("  - %s (%s)\n", l.Name, l.MainPath)
			}
			fmt.Printf("Prices:            %s\n", p.PricesPath)
			fmt.Println("CUE:               Unified OK")
			return nil
		},
	}
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use: "check [ledger]", Short: "Validate ledger(s)", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withLedgers(args, func(l *engine.Ledger) error {
				fmt.Printf("== %s ==\n", l.Name)
				printDiags(l.Diags)
				if l.Diags.HasErrors() {
					return fmt.Errorf("check failed for %s", l.Name)
				}
				fmt.Println("OK")
				return nil
			})
		},
	}
}

func balancesCmd() *cobra.Command {
	var asOf string
	c := &cobra.Command{
		Use: "balances [ledger]", Short: "Balances as-of", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := engine.ParseDate(asOf)
			if err != nil {
				return err
			}
			if t.IsZero() {
				t = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
			}
			showLedger := len(args) == 0
			type row struct {
				ledger, account, amount, commodity string
			}
			var rows []row
			err = withLedgers(args, func(l *engine.Ledger) error {
				bals := l.BalancesAsOf(t)
				var accts []string
				for a := range bals {
					accts = append(accts, a)
				}
				sort.Strings(accts)
				for _, a := range accts {
					var cs []string
					for c := range bals[a] {
						cs = append(cs, c)
					}
					sort.Strings(cs)
					for _, c := range cs {
						rows = append(rows, row{
							ledger:    l.Name,
							account:   a,
							amount:    bals[a][c].FloatString(6),
							commodity: c,
						})
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			// Stable global order: account name, then commodity, then ledger.
			sort.Slice(rows, func(i, j int) bool {
				if rows[i].account != rows[j].account {
					return rows[i].account < rows[j].account
				}
				if rows[i].commodity != rows[j].commodity {
					return rows[i].commodity < rows[j].commodity
				}
				return rows[i].ledger < rows[j].ledger
			})
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			if showLedger {
				fmt.Fprintln(w, "LEDGER\tACCOUNT\tAMOUNT\tCOMMODITY")
				for _, r := range rows {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ledger, r.account, r.amount, r.commodity)
				}
			} else {
				fmt.Fprintln(w, "ACCOUNT\tAMOUNT\tCOMMODITY")
				for _, r := range rows {
					fmt.Fprintf(w, "%s\t%s\t%s\n", r.account, r.amount, r.commodity)
				}
			}
			return w.Flush()
		},
	}
	c.Flags().StringVar(&asOf, "as-of", "", "YYYY-MM-DD")
	return c
}

func journalCmd() *cobra.Command {
	var timeFilter, from, to string
	c := &cobra.Command{
		Use: "journal [ledger]", Short: "Journal", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolvePeriod(timeFilter, from, to)
			if err != nil {
				return err
			}
			return withLedgers(args, func(l *engine.Ledger) error {
				fmt.Printf("== %s ==", l.Name)
				if !r.Empty() {
					fmt.Printf("  [%s]", r.Label())
				}
				fmt.Println()
				for _, e := range l.Journal(r.Start, r.End) {
					switch e.Kind {
					case "txn":
						fmt.Printf("%s * %q\n", e.Date.Format("2006-01-02"), e.Narration)
						for _, p := range e.Postings {
							if p.Units == nil || p.Units.Commodity == "" && p.Units.Number.Sign() == 0 {
								fmt.Printf("  %s\n", p.Account)
								continue
							}
							fmt.Printf("  %-40s %s %s\n", p.Account, p.Units.Number.FloatString(4), p.Units.Commodity)
						}
					case "note":
						fmt.Printf("%s note %s %q\n", e.Date.Format("2006-01-02"), e.Account, e.Comment)
					case "event":
						fmt.Printf("%s event %q %q\n", e.Date.Format("2006-01-02"), e.Narration, e.Comment)
					}
				}
				return nil
			})
		},
	}
	addTimeFlags(c, &timeFilter, &from, &to)
	return c
}

func pnlCmd() *cobra.Command {
	var timeFilter, from, to string
	c := &cobra.Command{
		Use: "pnl [ledger]", Short: "P&L for a Fava-style period", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolvePeriod(timeFilter, from, to)
			if err != nil {
				return err
			}
			return withLedgers(args, func(l *engine.Ledger) error {
				fmt.Printf("== %s ==", l.Name)
				if !r.Empty() {
					fmt.Printf("  [%s]", r.Label())
				}
				fmt.Println()
				p := l.PnL(r.Start, r.End)
				fmt.Println("Income:")
				var keys []string
				for k := range p.Income {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("  %-40s %s\n", k, p.Income[k].FloatString(4))
				}
				fmt.Println("Expenses:")
				keys = nil
				for k := range p.Expenses {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("  %-40s %s\n", k, p.Expenses[k].FloatString(4))
				}
				return nil
			})
		},
	}
	addTimeFlags(c, &timeFilter, &from, &to)
	return c
}

// addTimeFlags registers Fava-style --time plus optional --from/--to overrides.
func addTimeFlags(c *cobra.Command, timeFilter, from, to *string) {
	c.Flags().StringVar(timeFilter, "time", "", "Fava-style period: 2024, 2024-03, 2024-Q1, month, month-1, year, 2020 - 2024-06")
	c.Flags().StringVar(from, "from", "", "inclusive start YYYY-MM-DD (overrides --time start if set alone with --to)")
	c.Flags().StringVar(to, "to", "", "inclusive end YYYY-MM-DD")
}

// resolvePeriod prefers --time; if empty, uses --from/--to; if both empty, all time.
func resolvePeriod(timeFilter, from, to string) (period.Range, error) {
	if timeFilter != "" {
		if from != "" || to != "" {
			return period.Range{}, fmt.Errorf("use either --time or --from/--to, not both")
		}
		return period.Parse(timeFilter, time.Now())
	}
	f, err := engine.ParseDate(from)
	if err != nil {
		return period.Range{}, err
	}
	t, err := engine.ParseDate(to)
	if err != nil {
		return period.Range{}, err
	}
	raw := ""
	if from != "" || to != "" {
		raw = from + " … " + to
	}
	return period.Range{Start: f, End: t, Raw: raw}, nil
}

func networthCmd() *cobra.Command {
	var asOf string
	c := &cobra.Command{
		Use: "networth [ledger]", Short: "Net worth", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := engine.ParseDate(asOf)
			if err != nil {
				return err
			}
			if t.IsZero() {
				t = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
			}
			return withLedgers(args, func(l *engine.Ledger) error {
				lines, total, err := l.NetWorth(t)
				if err != nil {
					return err
				}
				fmt.Printf("== %s net worth (%s) ==\n", l.Name, l.OpCurrency)
				for _, ln := range lines {
					flag := ""
					if ln.UsedCost {
						flag = " (cost)"
					}
					fmt.Printf("  %-40s %12s %s => %s %s%s\n",
						ln.Account, ln.Units.FloatString(4), ln.Commodity, ln.Value.FloatString(2), l.OpCurrency, flag)
				}
				fmt.Printf("TOTAL %s %s\n", total.FloatString(2), l.OpCurrency)
				return nil
			})
		},
	}
	c.Flags().StringVar(&asOf, "as-of", "", "YYYY-MM-DD")
	return c
}

func accountCmd() *cobra.Command {
	var timeFilter, from, to string
	c := &cobra.Command{
		Use:   "account <ledger> <account>",
		Short: "Show one account (balance, period change, journal)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolvePeriod(timeFilter, from, to)
			if err != nil {
				return err
			}
			p, pdb, pdiags, err := engine.OpenProject(mustCwd())
			if err != nil {
				return err
			}
			printDiags(pdiags)
			l, err := engine.OpenLedger(p, pdb, args[0])
			if err != nil {
				return err
			}
			acct := args[1]
			asOf := r.End
			if asOf.IsZero() {
				asOf = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
			}
			fmt.Printf("== %s · %s ==", l.Name, acct)
			if !r.Empty() {
				fmt.Printf("  [%s]", r.Label())
			}
			fmt.Println()
			fmt.Println("Balance:")
			bals := l.AccountBalances(acct, asOf)
			if len(bals) == 0 {
				fmt.Println("  (zero)")
			} else {
				var cs []string
				for c := range bals {
					cs = append(cs, c)
				}
				sort.Strings(cs)
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				for _, c := range cs {
					fmt.Fprintf(w, "  %s\t%s\n", bals[c].FloatString(6), c)
				}
				w.Flush()
			}
			fmt.Println("Change in period:")
			act := l.AccountActivity(acct, r.Start, r.End)
			if len(act) == 0 {
				fmt.Println("  (none)")
			} else {
				var cs []string
				for c := range act {
					cs = append(cs, c)
				}
				sort.Strings(cs)
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				for _, c := range cs {
					fmt.Fprintf(w, "  %s\t%s\n", act[c].FloatString(6), c)
				}
				w.Flush()
			}
			fmt.Println("Journal:")
			for _, e := range l.JournalForAccount(acct, r.Start, r.End) {
				if e.Kind != "txn" {
					fmt.Printf("%s %s %s\n", e.Date.Format("2006-01-02"), e.Kind, e.Comment)
					continue
				}
				fmt.Printf("%s * %q\n", e.Date.Format("2006-01-02"), e.Narration)
				for _, p := range e.Postings {
					mark := "  "
					if p.Account == acct {
						mark = "* "
					}
					if p.Units == nil {
						fmt.Printf("%s%s\n", mark, p.Account)
						continue
					}
					fmt.Printf("%s%-40s %s %s\n", mark, p.Account, p.Units.Number.FloatString(4), p.Units.Commodity)
				}
			}
			return nil
		},
	}
	addTimeFlags(c, &timeFilter, &from, &to)
	return c
}

func parseCmd() *cobra.Command {
	return &cobra.Command{
		Use: "parse <file>", Short: "Dump directives from a file", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			dirs, diags, err := parser.Parse(args[0], src)
			printDiags(diags)
			if err != nil {
				return err
			}
			for _, d := range dirs {
				fmt.Printf("%T date=%s\n", d, d.GetDate().Format("2006-01-02"))
			}
			return nil
		},
	}
}

func webCmd() *cobra.Command {
	var addr string
	c := &cobra.Command{
		Use: "web [ledger]", Short: "Read-only web UI", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, pdb, _, err := engine.OpenProject(mustCwd())
			if err != nil {
				return err
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return web.Listen(p, pdb, name, addr)
		},
	}
	c.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "listen address (host:port)")
	return c
}
