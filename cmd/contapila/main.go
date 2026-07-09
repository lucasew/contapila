package main

import (
	"fmt"
	"os"

	"github.com/contapila/contapila/internal/ledger"
	"github.com/contapila/contapila/internal/project"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "contapila",
	Short: "Contapila is a self-contained Beancount-class ledger engine",
}

var checkCmd = &cobra.Command{
	Use:   "check [ledger]",
	Short: "Validate ledger configuration and includes",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		proj, err := project.Discover(cwd)
		if err != nil {
			return err
		}

		var ledgers []string
		if len(args) > 0 {
			ledgers = []string{args[0]}
		} else {
			ledgers, err = proj.ListLedgers()
			if err != nil {
				return err
			}
			if len(ledgers) == 0 {
				return fmt.Errorf("no ledgers found in project %s", proj.Root)
			}
		}

		hasError := false
		for _, name := range ledgers {
			path, err := proj.LedgerPath(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				hasError = true
				continue
			}

			loader := ledger.NewLoader()
			l, err := loader.Load(path, proj.Root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading ledger %q: %v\n", name, err)
				hasError = true
				continue
			}

			fmt.Printf("Ledger %q: OK (%d directives, %d accounts)\n", name, len(l.Directives), len(l.Config.Accounts))
		}

		if hasError {
			os.Exit(1)
		}
		return nil
	},
}

func main() {
	rootCmd.AddCommand(checkCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
