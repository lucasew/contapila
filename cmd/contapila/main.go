package main

import (
	"fmt"
	"log"
	"os"

	"github.com/contapila/contapila/pkg/core"
	"github.com/contapila/contapila/pkg/ledger"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "contapila"}

	var journalCmd = &cobra.Command{
		Use:   "journal [file]",
		Short: "Show journal",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			l, err := ledger.Load(args[0])
			if err != nil {
				log.Fatal(err)
			}

			for _, d := range l.Stream {
				dateStr := d.GetDate().Format("2006-01-02")
				switch v := d.(type) {
				case *core.Transaction:
					fmt.Printf("%s %s \"%s\" \"%s\"\n", dateStr, v.Flag, v.Payee, v.Narration)
					for _, p := range v.Postings {
						fmt.Printf("  %-30s %10s %s\n", p.Account, p.Amount, p.Currency)
					}
				case core.Note:
					fmt.Printf("%s note %s \"%s\"\n", dateStr, v.Account, v.Comment)
				case core.Event:
					fmt.Printf("%s event \"%s\" \"%s\"\n", dateStr, v.Type, v.Description)
				}
				fmt.Println()
			}
		},
	}

	rootCmd.AddCommand(journalCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
