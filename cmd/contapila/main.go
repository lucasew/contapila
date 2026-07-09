package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/contapila/contapila/internal/adapter"
	"github.com/contapila/contapila/internal/directive"
	"github.com/spf13/cobra"
)

func main() {
	var jsonOutput bool

	rootCmd := &cobra.Command{
		Use:   "contapila",
		Short: "Contapila is a Beancount-class ledger engine",
	}

	parseCmd := &cobra.Command{
		Use:   "parse [file]",
		Short: "Parse a Beancount file and print a summary of directives",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			src, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			directives, diagnostics, err := adapter.Parse(filename, src)
			for _, diag := range diagnostics {
				fmt.Fprintln(os.Stderr, diag.String())
			}

			if err != nil {
				return fmt.Errorf("parse failed: %w", err)
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(directives)
			}

			for _, d := range directives {
				printSummary(d)
			}

			return nil
		},
	}

	parseCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	rootCmd.AddCommand(parseCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printSummary(d directive.Directive) {
	switch v := d.(type) {
	case *directive.Transaction:
		fmt.Printf("%s * %q %q\n", v.Date.Format("2006-01-02"), v.Payee, v.Narration)
		for _, p := range v.Postings {
			fmt.Printf("  %s\n", p.Account)
		}
	case *directive.Open:
		fmt.Printf("%s open %s\n", v.Date.Format("2006-01-02"), v.Account)
	case *directive.Close:
		fmt.Printf("%s close %s\n", v.Date.Format("2006-01-02"), v.Account)
	case *directive.Commodity:
		fmt.Printf("%s commodity %s\n", v.Date.Format("2006-01-02"), v.Currency)
	case *directive.MarketPrice:
		fmt.Printf("%s price %s %s %s\n", v.Date.Format("2006-01-02"), v.Commodity, v.Amount.Number.String(), v.Amount.Commodity)
	case *directive.Balance:
		fmt.Printf("%s balance %s %s %s\n", v.Date.Format("2006-01-02"), v.Account, v.Amount.Number.String(), v.Amount.Commodity)
	case *directive.Pad:
		fmt.Printf("%s pad %s %s\n", v.Date.Format("2006-01-02"), v.Account, v.SourceAcc)
	case *directive.Note:
		fmt.Printf("%s note %s %q\n", v.Date.Format("2006-01-02"), v.Account, v.Comment)
	case *directive.Event:
		fmt.Printf("%s event %q %q\n", v.Date.Format("2006-01-02"), v.Type, v.Value)
	case *directive.Option:
		fmt.Printf("option %q %q\n", v.Name, v.Value)
	case *directive.Include:
		fmt.Printf("include %q\n", v.Path)
	case *directive.Unknown:
		fmt.Printf("unknown %s %q\n", v.Type, v.Text)
	default:
		fmt.Printf("unhandled directive type: %T\n", d)
	}
}
