package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show recent DNS query logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		logs, err := client.GetQueryLogs()
		if err != nil {
			return err
		}

		if len(logs) == 0 {
			fmt.Println("No query logs yet")
			return nil
		}

		fmt.Printf("%-40s %-16s %-10s %s\n", "DOMAIN", "CLIENT", "ACTION", "TIME")
		fmt.Println(strings.Repeat("-", 90))
		for _, l := range logs {
			domain := l.Domain
			if len(domain) > 38 {
				domain = domain[:35] + "..."
			}
			fmt.Printf("%-40s %-16s %-10s %s\n", domain, l.ClientIP, l.Action, l.Timestamp)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
