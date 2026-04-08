package cmd

import (
	"fmt"
	"os"

	mcpserver "github.com/hydradns/hydra-cli/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (JSON-RPC 2.0 over stdio)",
	Long:  "Starts a Model Context Protocol server that exposes HydraDNS tools for AI assistants. Communicates via JSON-RPC 2.0 over stdin/stdout.",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcpserver.NewServer(client)
		if err := server.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
