package main

import (
	"contapila/internal/booking"
	"contapila/internal/parser"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "contapila",
	Short: "Contapila is a ledger tool",
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

func main() {
	rootCmd.AddCommand(checkCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
