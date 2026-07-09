package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/lucasew/contapila-go/internal/ledger"
	"github.com/lucasew/contapila-go/internal/model"
	"github.com/lucasew/contapila-go/internal/price"
	"github.com/lucasew/contapila-go/pkg/project"
	"github.com/spf13/cobra"
)

var asOfFlag string

var rootCmd = &cobra.Command{
	Use:   "contapila",
	Short: "Contapila is a self-contained Beancount-class ledger engine",
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"doctor"},
	Short:   "Show project status and verify layout",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		p, err := project.OpenProject(cwd)
		if err != nil {
			return err
		}

		fmt.Printf("Project root:      %s\n", p.Root)
		fmt.Printf("contapila.cue:     %s\n", filepath.Join(p.Root, "contapila.cue"))

		fmt.Printf("Ledgers (%d):\n", len(p.Ledgers))
		if len(p.Ledgers) == 0 {
			return fmt.Errorf("zero ledgers found")
		}
		for _, l := range p.Ledgers {
			fmt.Printf("  - %s (%s)\n", l.Name, l.MainPath)
		}

		fmt.Printf("Prices:            %s\n", p.PricesPath)
		if p.PricesMissing {
			fmt.Println("  (missing - warning was emitted)")
		} else if p.PricesEmpty {
			fmt.Println("  (empty - warning was emitted)")
		} else {
			fmt.Println("  (found)")
		}

		fmt.Println("CUE:               Unified OK")

		return nil
	},
}

var networthCmd = &cobra.Command{
	Use:   "networth [ledger]",
	Short: "Calculate net worth",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		p, err := project.OpenProject(cwd)
		if err != nil {
			return err
		}

		asOf := time.Now()
		if asOfFlag != "" {
			asOf, err = time.Parse("2006-01-02", asOfFlag)
			if err != nil {
				return fmt.Errorf("invalid date format: %w", err)
			}
		}

		// Parser is missing per SPEC §5.3.
		// This command structure is ready but data loading will be empty until the grammar lands.
		var priceDirectives []model.Directive
		db := price.NewPriceDB(priceDirectives)

		ledgersToReport := p.Ledgers
		if len(args) > 0 {
			name := args[0]
			var found bool
			for _, l := range p.Ledgers {
				if l.Name == name {
					ledgersToReport = []project.Ledger{l}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("ledger %q not found", name)
			}
		}

		var cueOperatingCurrencies []string
		oc := p.Config.Value.LookupPath(cue.ParsePath("operating_currency"))
		if oc.Exists() {
			oc.Decode(&cueOperatingCurrencies)
		}

		for _, leg := range ledgersToReport {
			fmt.Printf("--- Ledger: %s ---\n", leg.Name)

			// Data loading is deferred until modernc grammar is available.
			var directives []model.Directive
			l := &ledger.Ledger{Name: leg.Name, Directives: directives}

			opCurr, explicit := l.ResolveOperatingCurrency(cueOperatingCurrencies)
			if opCurr == "" {
				fmt.Fprintf(os.Stderr, "Error: cannot determine operating currency for ledger %q\n", leg.Name)
				fmt.Fprintln(os.Stderr, "Note: Beancount parser is not yet implemented (waiting for tree-sitter grammar).")
				continue
			}
			if !explicit {
				fmt.Fprintf(os.Stderr, "Warning: inferred operating currency %q for ledger %q\n", opCurr, leg.Name)
			}

			positions := l.GetPositions(asOf)
			fmt.Printf("%-30s %15s %15s\n", "Account", "Amount", "Converted ("+opCurr+")")
			fmt.Println(strings.Repeat("-", 62))

			for _, pos := range positions {
				conv, _, _ := ledger.ConvertPosition(pos, db, opCurr, asOf)
				fmt.Printf("%-30s %10s %-4s %15s\n",
					pos.Account,
					pos.Units.FloatString(2),
					pos.Commodity,
					conv.FloatString(2))
			}
			fmt.Println(strings.Repeat("-", 62))

			nw := ledger.CalculateNetWorth(positions, db, opCurr, asOf)
			for _, w := range nw.Warnings {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
			}

			fmt.Printf("Total Net Worth: %s %s\n", nw.Total.FloatString(2), nw.Currency)
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	networthCmd.Flags().StringVar(&asOfFlag, "as-of", "", "Calculate net worth as of this date (YYYY-MM-DD); defaults to today")
	rootCmd.AddCommand(networthCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
