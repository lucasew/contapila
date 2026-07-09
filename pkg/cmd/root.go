package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/lucasew/contapila/pkg/ledger"
	"github.com/lucasew/contapila/pkg/web"
	"github.com/spf13/cobra"
)

var (
	port    int
	asOfStr string
)

var rootCmd = &cobra.Command{
	Use:   "contapila",
	Short: "Contapila is a self-contained ledger engine and web UI",
}

var checkCmd = &cobra.Command{
	Use:   "check [ledger]",
	Short: "Validate the ledger",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, err := ledger.DiscoverProject(".")
		if err != nil {
			return err
		}

		ledgers := args
		if len(ledgers) == 0 {
			ledgers = proj.LedgerNames
		}

		for _, name := range ledgers {
			l, err := proj.LoadLedger(name)
			if err != nil {
				fmt.Printf("Error loading ledger %s: %v\n", name, err)
				continue
			}
			diagnostics, err := l.Check()
			if err != nil {
				return err
			}
			fmt.Printf("--- Ledger: %s ---\n", name)
			if len(diagnostics) == 0 {
				fmt.Println("Clean")
			} else {
				for _, d := range diagnostics {
					fmt.Println(d.String())
				}
			}
		}
		return nil
	},
}

var balancesCmd = &cobra.Command{
	Use:   "balances [ledger]",
	Short: "Show balances as-of date",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("ledger name required")
		}
		name := args[0]
		proj, err := ledger.DiscoverProject(".")
		if err != nil {
			return err
		}
		l, err := proj.LoadLedger(name)
		if err != nil {
			return err
		}

		asOf := time.Now()
		if asOfStr != "" {
			var err error
			asOf, err = time.Parse("2006-01-02", asOfStr)
			if err != nil {
				return fmt.Errorf("invalid as-of date: %v", err)
			}
		}

		balances, err := l.GetBalances(asOf)
		if err != nil {
			return err
		}
		for _, b := range balances {
			fmt.Printf("%-30s %10s %s\n", b.Account, b.Amount.FloatString(2), b.Commodity)
		}
		return nil
	},
}

var webCmd = &cobra.Command{
	Use:   "web [ledger]",
	Short: "Start the read-only web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		proj, err := ledger.DiscoverProject(".")
		if err != nil {
			return err
		}
		ledgerName := ""
		if len(args) > 0 {
			ledgerName = args[0]
		}
		return web.StartServer(proj, ledgerName, port)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(balancesCmd)
	rootCmd.AddCommand(webCmd)

	balancesCmd.Flags().StringVar(&asOfStr, "as-of", "", "Balances as of date (YYYY-MM-DD)")
	webCmd.Flags().IntVarP(&port, "port", "p", 5000, "Port to bind (default 5000)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
