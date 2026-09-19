package cmd

import (
	"fmt"
	"os"
	"strings"

	mcpserver "github.com/hydradns/hydra-cli/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpHTTPAddr  string
	mcpHTTPToken string
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (JSON-RPC 2.0 over stdio, or HTTP with --http)",
	Long: "Starts a Model Context Protocol server that exposes HydraDNS tools for AI assistants.\n\n" +
		"By default it communicates via JSON-RPC 2.0 over stdin/stdout. Pass --http to instead\n" +
		"serve the same tool set over HTTP for driving a managed fleet remotely; every HTTP\n" +
		"request must carry a bearer token. The HTTP transport carries management traffic only\n" +
		"(engine/policy control), never DNS query data.",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcpserver.NewServer(client)

		// Default (no --http): stdio transport, unchanged.
		if mcpHTTPAddr == "" {
			if err := server.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
				return err
			}
			return nil
		}

		// Optional HTTP transport.
		httpToken := mcpHTTPToken
		if httpToken == "" {
			httpToken = os.Getenv("HYDRA_MCP_TOKEN")
		}
		if httpToken == "" {
			return fmt.Errorf("--http requires a bearer token: set --http-token or HYDRA_MCP_TOKEN")
		}

		addr := normalizeHTTPAddr(mcpHTTPAddr)
		fmt.Fprintf(os.Stderr, "MCP HTTP transport listening on %s\n", addr)
		if err := server.RunHTTP(addr, httpToken); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			return err
		}
		return nil
	},
}

// normalizeHTTPAddr defaults a bare port (":7000") to loopback so the remote
// transport does not bind to all interfaces unless the operator opts in by
// specifying an explicit host (e.g. "0.0.0.0:7000").
func normalizeHTTPAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}

func init() {
	mcpCmd.Flags().StringVar(&mcpHTTPAddr, "http", "", "Serve MCP over HTTP JSON-RPC on this address (e.g. :7000). Omit for stdio. A bare :port binds to localhost.")
	mcpCmd.Flags().StringVar(&mcpHTTPToken, "http-token", "", "Bearer token required for the HTTP transport (or set HYDRA_MCP_TOKEN)")
	rootCmd.AddCommand(mcpCmd)
}
