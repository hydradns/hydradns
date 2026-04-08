package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "List DNS policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := client.ListPolicies()
		if err != nil {
			return err
		}

		fmt.Printf("Policies: %d total (%d active, %d inactive)\n\n",
			data.TotalPolicies, data.ActivePolicies, data.InactivePolicies)

		for _, p := range data.List {
			domains := strings.Join(p.Domains, ", ")
			if len(domains) > 50 {
				domains = domains[:47] + "..."
			}
			fmt.Printf("  %-25s %-8s pri=%-4d %s\n", p.ID, p.Action, p.Priority, domains)
		}
		return nil
	},
}

var policiesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeletePolicy(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted policy: %s\n", args[0])
		return nil
	},
}

func init() {
	policiesCmd.AddCommand(policiesDeleteCmd)
	rootCmd.AddCommand(policiesCmd)
}
