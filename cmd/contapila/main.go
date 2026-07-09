package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"contapila/internal/ledger"
	"contapila/internal/model"
	"contapila/internal/parser"
	"contapila/internal/price"
	"contapila/internal/project"

	"github.com/spf13/cobra"
)

var asOfFlag string

func main() {
	var rootCmd = &cobra.Command{Use: "contapila"}

	var networthCmd = &cobra.Command{
		Use:   "networth [ledger]",
		Short: "Calculate net worth",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNetworth,
	}

	networthCmd.Flags().StringVar(&asOfFlag, "as-of", "", "Calculate net worth as of this date (YYYY-MM-DD); defaults to today")

	rootCmd.AddCommand(networthCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runNetworth(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	root, err := project.FindRoot(cwd)
	if err != nil {
		return err
	}

	p, err := project.LoadProject(root)
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

	// Load shared prices
	pricesPath := fmt.Sprintf("%s/prices.beancount", root)
	var priceDirectives []model.Directive
	if _, err := os.Stat(pricesPath); err == nil {
		priceDirectives, err = parser.LoadFiles(pricesPath, make(map[string]bool))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load prices.beancount: %v\n", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Warning: prices.beancount missing or empty")
	}
	db := price.NewPriceDB(priceDirectives)

	ledgersToReport := p.Ledgers
	if len(args) > 0 {
		name := args[0]
		path, ok := p.Ledgers[name]
		if !ok {
			return fmt.Errorf("ledger %q not found", name)
		}
		ledgersToReport = map[string]string{name: path}
	}

	for name, path := range ledgersToReport {
		fmt.Printf("--- Ledger: %s ---\n", name)
		directives, err := parser.LoadFiles(path, make(map[string]bool))
		if err != nil {
			return fmt.Errorf("failed to load ledger %q: %w", name, err)
		}

		l := &ledger.Ledger{Name: name, Directives: directives}
		opCurr, explicit := l.ResolveOperatingCurrency()
		if opCurr == "" {
			fmt.Fprintf(os.Stderr, "Error: cannot determine operating currency for ledger %q\n", name)
			continue
		}
		if !explicit {
			fmt.Fprintf(os.Stderr, "Warning: inferred operating currency %q for ledger %q\n", opCurr, name)
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
}
