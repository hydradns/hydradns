package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DNS engine status and query statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, err := client.GetDashboardSummary()
		if err != nil {
			return err
		}
		engine, err := client.GetEngineStatus()
		if err != nil {
			return err
		}

		fmt.Println("=== HydraDNS Status ===")
		fmt.Println()

		status := "STOPPED"
		if engine.AcceptingQueries {
			status = "RUNNING"
		}
		fmt.Printf("  Engine:     %s\n", status)
		fmt.Printf("  Enabled:    %v\n", engine.Enabled)
		if engine.LastError != "" {
			fmt.Printf("  Last Error: %s\n", engine.LastError)
		}
		fmt.Println()
		fmt.Printf("  Total Queries:  %d\n", summary.TotalQueries)
		fmt.Printf("  Blocked:        %d\n", summary.BlockedQueries)
		fmt.Printf("  Allowed:        %d\n", summary.AllowedQueries)
		fmt.Printf("  Redirected:     %d\n", summary.RedirectedQueries)
		fmt.Printf("  Block Rate:     %.1f%%\n", summary.BlockRatePercent)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
