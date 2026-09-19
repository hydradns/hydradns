package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blocklistsCmd = &cobra.Command{
	Use:   "blocklists",
	Short: "List blocklist sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := client.ListBlocklists()
		if err != nil {
			return err
		}

		fmt.Printf("Blocklists: %d sources, %d domains total\n\n",
			data.TotalBlocklists, data.TotalDomains)

		for _, b := range data.ActiveLists {
			status := "active"
			if !b.Enabled {
				status = "disabled"
			}
			fmt.Printf("  %-20s %-10s %8d domains  [%s]\n", b.ID, b.Format, b.DomainsCount, status)
		}
		return nil
	},
}

var (
	blAddID       string
	blAddName     string
	blAddURL      string
	blAddFormat   string
	blAddCategory string
)

var blocklistsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a blocklist source",
	RunE: func(cmd *cobra.Command, args []string) error {
		req := map[string]interface{}{
			"id":     blAddID,
			"name":   blAddName,
			"url":    blAddURL,
			"format": blAddFormat,
		}
		if blAddCategory != "" {
			req["category"] = blAddCategory
		}

		bl, err := client.CreateBlocklist(req)
		if err != nil {
			return err
		}
		fmt.Printf("Added blocklist: %s (%s)\n", bl.ID, bl.URL)
		return nil
	},
}

var blocklistsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a blocklist source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteBlocklist(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted blocklist: %s\n", args[0])
		return nil
	},
}

func init() {
	blocklistsAddCmd.Flags().StringVar(&blAddID, "id", "", "Blocklist ID (required)")
	blocklistsAddCmd.Flags().StringVar(&blAddName, "name", "", "Display name (required)")
	blocklistsAddCmd.Flags().StringVar(&blAddURL, "url", "", "Source URL (required)")
	blocklistsAddCmd.Flags().StringVar(&blAddFormat, "format", "hosts", "Format: hosts, domains, adblock")
	blocklistsAddCmd.Flags().StringVar(&blAddCategory, "category", "", "Category")
	blocklistsAddCmd.MarkFlagRequired("id")
	blocklistsAddCmd.MarkFlagRequired("name")
	blocklistsAddCmd.MarkFlagRequired("url")

	blocklistsCmd.AddCommand(blocklistsAddCmd)
	blocklistsCmd.AddCommand(blocklistsDeleteCmd)
	rootCmd.AddCommand(blocklistsCmd)
}
