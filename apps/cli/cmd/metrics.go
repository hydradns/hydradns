package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show DNS query performance metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := client.GetMetrics()
		if err != nil {
			return err
		}

		fmt.Printf("=== DNS Metrics (%ds window) ===\n\n", m.WindowSeconds)
		fmt.Printf("  Queries:    %d\n", m.Queries.Total)
		fmt.Printf("  Errors:     %d (%.2f%%)\n", m.Queries.Errors, m.Queries.ErrorRate*100)
		fmt.Printf("  Grade:      %s\n\n", m.Grade)
		fmt.Printf("  Latency p50: %dms\n", m.LatencyMs.P50)
		fmt.Printf("  Latency p95: %dms\n", m.LatencyMs.P95)
		fmt.Printf("  Latency p99: %dms\n", m.LatencyMs.P99)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}
