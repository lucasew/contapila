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
	root.AddCommand(statusCmd(), checkCmd(), balancesCmd(), journalCmd(), pnlCmd(), networthCmd(), parseCmd(), webCmd())
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
	var from, to string
	c := &cobra.Command{
		Use: "journal [ledger]", Short: "Journal", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := engine.ParseDate(from)
			if err != nil {
				return err
			}
			t, err := engine.ParseDate(to)
			if err != nil {
				return err
			}
			return withLedgers(args, func(l *engine.Ledger) error {
				fmt.Printf("== %s ==\n", l.Name)
				for _, e := range l.Journal(f, t) {
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
	c.Flags().StringVar(&from, "from", "", "YYYY-MM-DD")
	c.Flags().StringVar(&to, "to", "", "YYYY-MM-DD")
	return c
}

func pnlCmd() *cobra.Command {
	var from, to string
	c := &cobra.Command{
		Use: "pnl [ledger]", Short: "P&L", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := engine.ParseDate(from)
			if err != nil {
				return err
			}
			t, err := engine.ParseDate(to)
			if err != nil {
				return err
			}
			return withLedgers(args, func(l *engine.Ledger) error {
				fmt.Printf("== %s ==\n", l.Name)
				p := l.PnL(f, t)
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
	c.Flags().StringVar(&from, "from", "", "YYYY-MM-DD")
	c.Flags().StringVar(&to, "to", "", "YYYY-MM-DD")
	return c
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
