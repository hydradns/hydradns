package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hydradns/hydra-cli/api"
)

// JSON-RPC 2.0 types

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// MarshalJSON ensures only result OR error is present, never both or neither
func (r Response) MarshalJSON() ([]byte, error) {
	type Alias Response
	if r.Error != nil {
		return json.Marshal(struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
			Error   *Error `json:"error"`
		}{r.JSONRPC, r.ID, r.Error})
	}
	return json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Result  any    `json:"result"`
	}{r.JSONRPC, r.ID, r.Result})
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP protocol types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
}

type Tool struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema InputSchema      `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ToolAnnotations are optional behavioural hints attached to a tool so MCP
// clients can distinguish read-only tools from destructive ones and warn (or
// require confirmation) before running a mutating operation.
type ToolAnnotations struct {
	ReadOnlyHint         bool `json:"readOnlyHint,omitempty"`
	DestructiveHint      bool `json:"destructiveHint,omitempty"`
	ConfirmationRequired bool `json:"confirmationRequired,omitempty"`
}

type InputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ArrayProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Items       ItemType `json:"items"`
}

type ItemType struct {
	Type string `json:"type"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Server

type Server struct {
	client apiClient
	role   Role
}

// NewServer constructs an MCP server, resolving its permission role from the
// MCP_ROLE environment variable. When MCP_ROLE is unset the server runs as
// admin (no restriction), preserving prior behaviour.
func NewServer(client *api.Client) *Server {
	return &Server{client: client, role: resolveRole(os.Getenv("MCP_ROLE"))}
}

// NewServerWithRole constructs a server with an explicit role, bypassing the
// MCP_ROLE environment lookup. Primarily useful for tests.
func NewServerWithRole(client *api.Client, role Role) *Server {
	return &Server{client: client, role: role}
}

func (s *Server) tools() []Tool {
	list := []Tool{
		{
			Name:        "get_status",
			Description: "Get DNS engine status and query statistics",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "toggle_engine",
			Description: "Enable or disable the DNS engine",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"enabled": Property{Type: "boolean", Description: "true to enable, false to disable"},
				},
				Required: []string{"enabled"},
			},
		},
		{
			Name:        "block_domain",
			Description: "Block a domain by creating a block policy",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"domain": Property{Type: "string", Description: "Domain name to block (e.g. ads.example.com)"},
				},
				Required: []string{"domain"},
			},
		},
		{
			Name:        "create_policy",
			Description: "Create a DNS policy to block, allow, or redirect multiple domains in one rule. Use this instead of block_domain when handling multiple domains (e.g. 'block all social media').",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"name":     Property{Type: "string", Description: "Human-readable policy name (e.g. 'Block Social Media')"},
					"action":   Property{Type: "string", Description: "BLOCK, ALLOW, or REDIRECT"},
					"domains":  ArrayProperty{Type: "array", Description: "List of domains to apply the policy to", Items: ItemType{Type: "string"}},
					"priority": Property{Type: "integer", Description: "Priority (higher wins). Default 100."},
				},
				Required: []string{"name", "action", "domains"},
			},
		},
		{
			Name:        "unblock_domain",
			Description: "Remove a block policy by its ID",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"policy_id": Property{Type: "string", Description: "The policy ID to remove"},
				},
				Required: []string{"policy_id"},
			},
		},
		{
			Name:        "list_policies",
			Description: "List all DNS policies",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "list_blocklists",
			Description: "List blocklist sources and domain counts",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "get_query_logs",
			Description: "Get recent DNS query logs",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "get_metrics",
			Description: "Get DNS query performance metrics including latency percentiles",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "get_weekly_summary",
			Description: "Get a natural-language rollup of DNS security activity (traffic, block rate, performance, and protection coverage) built from live stats, metrics, and query logs. Best for an at-a-glance report.",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "explain_anomaly",
			Description: "Inspect current DNS activity and describe anything unusual (elevated error rate, degraded latency, block-rate spikes, or a single client dominating traffic). Optionally pass a baseline to compare against.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"baseline_block_rate":     Property{Type: "number", Description: "Prior block rate percent to compare current block rate against (optional)"},
					"max_error_rate_percent":  Property{Type: "number", Description: "Error-rate threshold that counts as an anomaly. Default 5.0"},
					"block_rate_jump_percent": Property{Type: "number", Description: "Increase in block rate (percentage points) vs baseline that counts as a spike. Default 15.0"},
				},
			},
		},
		{
			Name:        "compare_to_last_month",
			Description: "Compare current DNS traffic and block rate against a baseline window (e.g. last month) and describe the changes in plain language. Supply the baseline figures from a prior summary.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"baseline_total_queries":   Property{Type: "integer", Description: "Total queries during the baseline period (optional)"},
					"baseline_blocked_queries": Property{Type: "integer", Description: "Blocked queries during the baseline period (optional)"},
					"baseline_block_rate":      Property{Type: "number", Description: "Block rate percent during the baseline period (optional)"},
					"label":                    Property{Type: "string", Description: "Name of the baseline period for the report. Default 'last month'"},
				},
			},
		},
		{
			Name:        "bulk_unblock",
			Description: "Remove multiple policies at once by their IDs. Reports which were removed and which failed.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"policy_ids": ArrayProperty{Type: "array", Description: "List of policy IDs to remove", Items: ItemType{Type: "string"}},
				},
				Required: []string{"policy_ids"},
			},
		},
		{
			Name:        "delete_policy",
			Description: "Delete a single DNS policy by its ID. Works for BLOCK, ALLOW, and REDIRECT policies.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"policy_id": Property{Type: "string", Description: "The policy ID to delete"},
				},
				Required: []string{"policy_id"},
			},
		},
	}

	// Attach behavioural annotations derived from the tool classification so
	// clients can flag read-only vs destructive (confirmation-required) tools.
	for i := range list {
		if isReadOnly(list[i].Name) {
			list[i].Annotations = &ToolAnnotations{ReadOnlyHint: true}
		} else {
			list[i].Annotations = &ToolAnnotations{DestructiveHint: true, ConfirmationRequired: true}
		}
	}
	return list
}

// Run serves the MCP protocol over stdin/stdout (the default transport).
func (s *Server) Run() error {
	return s.serve(os.Stdin, os.Stdout)
}

// serve runs the JSON-RPC read/dispatch loop over the given streams. It backs
// the stdio transport (Run) and is transport-agnostic so the request handling
// stays identical across transports.
func (s *Server) serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := out

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.writeResponse(writer, Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &Error{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		// Notifications (no ID) get no response
		if strings.HasPrefix(req.Method, "notifications/") {
			continue
		}

		resp := s.handleRequest(req)
		s.writeResponse(writer, resp)
	}
}

func (s *Server) writeResponse(w io.Writer, resp Response) {
	b, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s\n", b)
}

func (s *Server) handleRequest(req Request) Response {
	switch req.Method {
	case "initialize":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: map[string]any{
					"tools": map[string]any{},
				},
				ServerInfo: ServerInfo{
					Name:    "hydradns",
					Version: "1.0.0",
				},
			},
		}

	case "tools/list":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  ToolsListResult{Tools: s.tools()},
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &Error{Code: -32602, Message: "Invalid params"},
			}
		}
		// Role gate: reject tools the active role may not call, returning a
		// structured JSON-RPC error instead of executing the tool.
		if !s.role.Allows(params.Name) {
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   permissionError(s.role, params.Name),
			}
		}
		result := s.callTool(params)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	case "ping":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}

	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: -32601, Message: "Method not found: " + req.Method},
		}
	}
}

func (s *Server) callTool(params CallToolParams) CallToolResult {
	switch params.Name {
	case "get_status":
		return s.toolGetStatus()
	case "toggle_engine":
		return s.toolToggleEngine(params.Arguments)
	case "block_domain":
		return s.toolBlockDomain(params.Arguments)
	case "create_policy":
		return s.toolCreatePolicy(params.Arguments)
	case "unblock_domain":
		return s.toolUnblockDomain(params.Arguments)
	case "list_policies":
		return s.toolListPolicies()
	case "list_blocklists":
		return s.toolListBlocklists()
	case "get_query_logs":
		return s.toolGetQueryLogs()
	case "get_metrics":
		return s.toolGetMetrics()
	case "get_weekly_summary":
		return s.toolGetWeeklySummary()
	case "explain_anomaly":
		return s.toolExplainAnomaly(params.Arguments)
	case "compare_to_last_month":
		return s.toolCompareToLastMonth(params.Arguments)
	case "bulk_unblock":
		return s.toolBulkUnblock(params.Arguments)
	case "delete_policy":
		return s.toolDeletePolicy(params.Arguments)
	default:
		return CallToolResult{
			Content: []ContentItem{{Type: "text", Text: "Unknown tool: " + params.Name}},
			IsError: true,
		}
	}
}

func (s *Server) toolGetStatus() CallToolResult {
	engine, err := s.client.GetEngineStatus()
	if err != nil {
		return errorResult(err)
	}
	summary, err := s.client.GetDashboardSummary()
	if err != nil {
		return errorResult(err)
	}

	text := fmt.Sprintf("DNS Engine: enabled=%v, accepting_queries=%v\n"+
		"Queries: total=%d, blocked=%d, allowed=%d, block_rate=%.1f%%",
		engine.Enabled, engine.AcceptingQueries,
		summary.TotalQueries, summary.BlockedQueries, summary.AllowedQueries, summary.BlockRatePercent)

	return textResult(text)
}

func (s *Server) toolToggleEngine(args json.RawMessage) CallToolResult {
	var p struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}

	if err := s.client.ToggleEngine(p.Enabled); err != nil {
		return errorResult(err)
	}

	action := "disabled"
	if p.Enabled {
		action = "enabled"
	}
	return textResult(fmt.Sprintf("DNS engine %s", action))
}

func (s *Server) toolBlockDomain(args json.RawMessage) CallToolResult {
	var p struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}

	id := "cli-block-" + strings.ReplaceAll(p.Domain, ".", "-")
	_, err := s.client.CreatePolicy(map[string]interface{}{
		"id":       id,
		"name":     "MCP Block: " + p.Domain,
		"action":   "BLOCK",
		"domains":  []string{p.Domain},
		"priority": 150,
		"category": "mcp",
	})
	if err != nil {
		return errorResult(err)
	}

	return textResult(fmt.Sprintf("Blocked domain %s (policy: %s)", p.Domain, id))
}

func (s *Server) toolCreatePolicy(args json.RawMessage) CallToolResult {
	var p struct {
		Name     string   `json:"name"`
		Action   string   `json:"action"`
		Domains  []string `json:"domains"`
		Priority int      `json:"priority"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}

	if p.Priority == 0 {
		p.Priority = 100
	}

	// Generate a slug ID from the name
	id := "mcp-" + strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(p.Name, " ", "-"), "'", ""))

	_, err := s.client.CreatePolicy(map[string]interface{}{
		"id":       id,
		"name":     p.Name,
		"action":   strings.ToUpper(p.Action),
		"domains":  p.Domains,
		"priority": p.Priority,
		"category": "mcp",
	})
	if err != nil {
		return errorResult(err)
	}

	return textResult(fmt.Sprintf("Created policy '%s' (%s) with %d domains: %s",
		p.Name, strings.ToUpper(p.Action), len(p.Domains), strings.Join(p.Domains, ", ")))
}

func (s *Server) toolUnblockDomain(args json.RawMessage) CallToolResult {
	var p struct {
		PolicyID string `json:"policy_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}

	if err := s.client.DeletePolicy(p.PolicyID); err != nil {
		return errorResult(err)
	}

	return textResult(fmt.Sprintf("Removed policy: %s", p.PolicyID))
}

func (s *Server) toolListPolicies() CallToolResult {
	data, err := s.client.ListPolicies()
	if err != nil {
		return errorResult(err)
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	return textResult(string(b))
}

func (s *Server) toolListBlocklists() CallToolResult {
	data, err := s.client.ListBlocklists()
	if err != nil {
		return errorResult(err)
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	return textResult(string(b))
}

func (s *Server) toolGetQueryLogs() CallToolResult {
	logs, err := s.client.GetQueryLogs()
	if err != nil {
		return errorResult(err)
	}

	if len(logs) == 0 {
		return textResult("No query logs yet")
	}

	b, _ := json.MarshalIndent(logs, "", "  ")
	return textResult(string(b))
}

func (s *Server) toolGetMetrics() CallToolResult {
	m, err := s.client.GetMetrics()
	if err != nil {
		return errorResult(err)
	}

	text := fmt.Sprintf("Queries: total=%d, errors=%d, error_rate=%.2f%%\n"+
		"Latency: p50=%dms, p95=%dms, p99=%dms\n"+
		"Grade: %s",
		m.Queries.Total, m.Queries.Errors, m.Queries.ErrorRate*100,
		m.LatencyMs.P50, m.LatencyMs.P95, m.LatencyMs.P99,
		m.Grade)

	return textResult(text)
}

func textResult(text string) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: text}},
	}
}

func errorResult(err error) CallToolResult {
	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: err.Error()}},
		IsError: true,
	}
}
