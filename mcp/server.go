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

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]any    `json:"capabilities"`
	ServerInfo      ServerInfo        `json:"serverInfo"`
}

type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
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
	client *api.Client
}

func NewServer(client *api.Client) *Server {
	return &Server{client: client}
}

func (s *Server) tools() []Tool {
	return []Tool{
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
				Properties: map[string]Property{
					"enabled": {Type: "boolean", Description: "true to enable, false to disable"},
				},
				Required: []string{"enabled"},
			},
		},
		{
			Name:        "block_domain",
			Description: "Block a domain by creating a block policy",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"domain": {Type: "string", Description: "Domain name to block (e.g. ads.example.com)"},
				},
				Required: []string{"domain"},
			},
		},
		{
			Name:        "unblock_domain",
			Description: "Remove a block policy by its ID",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"policy_id": {Type: "string", Description: "The policy ID to remove"},
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
	}
}

func (s *Server) Run() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

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

	case "notifications/initialized":
		// No response for notifications
		return Response{JSONRPC: "2.0", ID: req.ID}

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
