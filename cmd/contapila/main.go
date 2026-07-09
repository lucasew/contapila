package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/lucasew/contapila-go/internal/booking"
	"github.com/lucasew/contapila-go/internal/parser"
	"github.com/lucasew/contapila-go/pkg/project"
)

var rootCmd = &cobra.Command{
	Use:   "contapila",
	Short: "Contapila is a self-contained Beancount-class ledger engine",
}

var checkCmd = &cobra.Command{
	Use:   "check <ledger>",
	Short: "Validate the ledger",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ledgerPath := args[0]
		f, err := os.Open(ledgerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening ledger: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		directives, err := parser.Parse(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing ledger: %v\n", err)
			os.Exit(1)
		}

		booker := booking.NewBooker()
		booker.Book(directives)

		hasErrors := false
		for _, diag := range booker.Diagnostics {
			severity := "WARNING"
			if diag.Severity == booking.Error {
				severity = "ERROR"
				hasErrors = true
			}
			fmt.Printf("%s: %s: %s\n", diag.Date.Format("2006-01-02"), severity, diag.Message)
		}

		if hasErrors {
			os.Exit(1)
		}
	},
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
			// According to SPEC, error if zero ledgers found when running status/check-style
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

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(checkCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
