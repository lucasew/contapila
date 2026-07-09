package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/lucasew/contapila-go/pkg/project"
)

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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
