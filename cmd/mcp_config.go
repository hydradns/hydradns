package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// mcpServerName is the key used for the HydraDNS entry in a client's
// mcpServers map.
const mcpServerName = "hydradns"

var mcpConfigClient string

// mcpServerEntry is a single stdio MCP server definition. The shape
// (command/args/env) is shared by Claude Desktop/Code, the Gemini CLI, and
// Cursor. Cursor additionally accepts an explicit transport "type".
type mcpServerEntry struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// mcpConfigFile is the top-level object every supported client expects.
type mcpConfigFile struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

// buildMCPConfig renders ready-to-paste MCP client configuration JSON for the
// given client. The emitted config points at the hydra binary located at
// exePath, invokes it as `hydra mcp`, and carries the current HydraDNS API URL
// and token as environment variables so the spawned server connects the same
// way this CLI does.
//
// All inputs are injected, so the function is pure and deterministic: tests can
// supply a fixed exe path and env without touching the real process.
func buildMCPConfig(clientName, exePath, apiURL, token string) (string, error) {
	entry := mcpServerEntry{
		Command: exePath,
		Args:    []string{"mcp"},
		Env: map[string]string{
			"HYDRA_API_URL": apiURL,
			"HYDRA_TOKEN":   token,
		},
	}

	switch clientName {
	case "claude", "gemini":
		// Claude Desktop/Code and the Gemini CLI both read the plain
		// mcpServers stdio shape (command/args/env).
	case "cursor":
		// Cursor marks stdio servers with an explicit transport type.
		entry.Type = "stdio"
	default:
		return "", fmt.Errorf("unknown client %q: supported clients are claude, gemini, cursor", clientName)
	}

	cfg := mcpConfigFile{
		MCPServers: map[string]mcpServerEntry{
			mcpServerName: entry,
		},
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// mcpConfigHint tells the user where the emitted config belongs for each client.
func mcpConfigHint(clientName string) string {
	switch clientName {
	case "claude":
		return "# Paste into claude_desktop_config.json (Claude Desktop) or .mcp.json in your project root (Claude Code)."
	case "gemini":
		return "# Paste into ~/.gemini/settings.json, or .gemini/settings.json in your project, for the Gemini CLI."
	case "cursor":
		return "# Paste into ~/.cursor/mcp.json (global) or .cursor/mcp.json (project) for Cursor."
	default:
		return ""
	}
}

var mcpConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Print ready-to-paste MCP client configuration",
	Long: "Prints MCP client configuration JSON that points at this hydra binary and\n" +
		"launches it as `hydra mcp`, carrying the current HYDRA_API_URL and HYDRA_TOKEN.\n\n" +
		"Use --client to target Claude Desktop/Code (default), the Gemini CLI, or Cursor.\n" +
		"The JSON is written to stdout; a hint about where to paste it goes to stderr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine hydra binary path: %w", err)
		}

		out, err := buildMCPConfig(mcpConfigClient, exe, apiURL, token)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), out)
		if hint := mcpConfigHint(mcpConfigClient); hint != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), hint)
		}
		return nil
	},
}

func init() {
	mcpConfigCmd.Flags().StringVar(&mcpConfigClient, "client", "claude", "target MCP client: claude, gemini, or cursor")
	mcpCmd.AddCommand(mcpConfigCmd)
}
