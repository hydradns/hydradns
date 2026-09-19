package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <domain> [domain...]",
	Short: "Block one or more domains",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domains := args
		id := "cli-block-" + strings.ReplaceAll(domains[0], ".", "-")

		_, err := client.CreatePolicy(map[string]interface{}{
			"id":       id,
			"name":     "CLI Block: " + strings.Join(domains, ", "),
			"action":   "BLOCK",
			"domains":  domains,
			"priority": 150,
			"category": "cli",
		})
		if err != nil {
			return err
		}

		fmt.Printf("Blocked: %s (policy: %s)\n", strings.Join(domains, ", "), id)
		return nil
	},
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <policy-id>",
	Short: "Remove a block policy by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeletePolicy(args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed policy: %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(unblockCmd)
}
