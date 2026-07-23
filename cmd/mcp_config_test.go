package cmd

import (
	"encoding/json"
	"testing"
)

func TestBuildMCPConfig_Clients(t *testing.T) {
	const (
		exePath = "/usr/local/bin/hydra"
		apiURL  = "http://192.168.1.10:8080"
		token   = "secret-token-123"
	)

	tests := []struct {
		client   string
		wantType string // expected transport "type"; "" means the field must be absent
	}{
		{"claude", ""},
		{"gemini", ""},
		{"cursor", "stdio"},
	}

	for _, tt := range tests {
		t.Run(tt.client, func(t *testing.T) {
			out, err := buildMCPConfig(tt.client, exePath, apiURL, token)
			if err != nil {
				t.Fatalf("buildMCPConfig(%q) returned error: %v", tt.client, err)
			}

			// Output must be valid JSON.
			var generic map[string]any
			if err := json.Unmarshal([]byte(out), &generic); err != nil {
				t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, out)
			}

			// Output must match the expected client config structure.
			var cfg mcpConfigFile
			if err := json.Unmarshal([]byte(out), &cfg); err != nil {
				t.Fatalf("output does not match config structure: %v", err)
			}

			entry, ok := cfg.MCPServers[mcpServerName]
			if !ok {
				t.Fatalf("mcpServers missing %q key; got %+v", mcpServerName, cfg.MCPServers)
			}

			if entry.Command != exePath {
				t.Errorf("command = %q, want %q", entry.Command, exePath)
			}
			if len(entry.Args) != 1 || entry.Args[0] != "mcp" {
				t.Errorf("args = %v, want [mcp]", entry.Args)
			}
			if got := entry.Env["HYDRA_API_URL"]; got != apiURL {
				t.Errorf("env HYDRA_API_URL = %q, want %q", got, apiURL)
			}
			if got := entry.Env["HYDRA_TOKEN"]; got != token {
				t.Errorf("env HYDRA_TOKEN = %q, want %q", got, token)
			}
			if entry.Type != tt.wantType {
				t.Errorf("type = %q, want %q", entry.Type, tt.wantType)
			}
		})
	}
}

func TestBuildMCPConfig_UnknownClient(t *testing.T) {
	if _, err := buildMCPConfig("vscode", "/usr/local/bin/hydra", "http://localhost:8080", ""); err == nil {
		t.Fatal("expected error for unknown client, got nil")
	}
}

func TestBuildMCPConfig_Deterministic(t *testing.T) {
	first, err := buildMCPConfig("claude", "/bin/hydra", "http://localhost:8080", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := buildMCPConfig("claude", "/bin/hydra", "http://localhost:8080", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Errorf("output not deterministic:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
