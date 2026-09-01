package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hydradns/hydra-cli/api"
)

// mockClient is a deterministic in-memory implementation of apiClient used to
// drive the MCP tools without a live control plane.
type mockClient struct {
	engine     *api.EngineStatus
	summary    *api.DashboardSummary
	metrics    *api.Metrics
	policies   *api.PolicyListData
	blocklists *api.BlocklistListData
	logs       []api.QueryLog

	deleted   []string
	deleteErr map[string]error
}

func (m *mockClient) GetEngineStatus() (*api.EngineStatus, error) { return m.engine, nil }
func (m *mockClient) GetDashboardSummary() (*api.DashboardSummary, error) {
	return m.summary, nil
}
func (m *mockClient) ToggleEngine(enabled bool) error { return nil }
func (m *mockClient) CreatePolicy(req map[string]interface{}) (*api.Policy, error) {
	return &api.Policy{}, nil
}
func (m *mockClient) DeletePolicy(id string) error {
	if err, ok := m.deleteErr[id]; ok {
		return err
	}
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockClient) ListPolicies() (*api.PolicyListData, error)      { return m.policies, nil }
func (m *mockClient) ListBlocklists() (*api.BlocklistListData, error) { return m.blocklists, nil }
func (m *mockClient) GetQueryLogs() ([]api.QueryLog, error)           { return m.logs, nil }
func (m *mockClient) GetMetrics() (*api.Metrics, error)               { return m.metrics, nil }

// seededMock returns a mock loaded with representative, deterministic data.
func seededMock() *mockClient {
	m := &api.Metrics{Grade: "A"}
	m.Queries.Total = 10000
	m.Queries.Errors = 30
	m.Queries.ErrorRate = 0.003
	m.LatencyMs.P50 = 1
	m.LatencyMs.P95 = 8
	m.LatencyMs.P99 = 20

	return &mockClient{
		engine:     &api.EngineStatus{Enabled: true, AcceptingQueries: true},
		summary:    &api.DashboardSummary{TotalQueries: 10000, BlockedQueries: 2000, AllowedQueries: 8000, RedirectedQueries: 0, BlockRatePercent: 20.0},
		metrics:    m,
		policies:   &api.PolicyListData{TotalPolicies: 5, ActivePolicies: 4, InactivePolicies: 1},
		blocklists: &api.BlocklistListData{TotalBlocklists: 3, TotalDomains: 120000},
		// Balanced client distribution (no client > 50%) so the baseline
		// case reports no anomaly; ads.example.com is the top blocked domain.
		logs: []api.QueryLog{
			{Domain: "ads.example.com", ClientIP: "10.0.0.5", Action: "BLOCK"},
			{Domain: "ads.example.com", ClientIP: "10.0.0.6", Action: "BLOCK"},
			{Domain: "good.example.com", ClientIP: "10.0.0.7", Action: "ALLOW"},
			{Domain: "good.example.com", ClientIP: "10.0.0.8", Action: "ALLOW"},
			{Domain: "other.example.com", ClientIP: "10.0.0.5", Action: "ALLOW"},
			{Domain: "another.example.com", ClientIP: "10.0.0.6", Action: "ALLOW"},
		},
	}
}

func newTestServer(m *mockClient) *Server { return &Server{client: m} }

// --- registration ---

func TestToolsListIncludesNewTools(t *testing.T) {
	s := newTestServer(seededMock())
	got := map[string]Tool{}
	for _, tl := range s.tools() {
		got[tl.Name] = tl
	}

	want := []string{
		"get_weekly_summary", "explain_anomaly", "compare_to_last_month",
		"bulk_unblock", "delete_policy",
	}
	for _, name := range want {
		tl, ok := got[name]
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if tl.Description == "" {
			t.Errorf("tool %q has empty description", name)
		}
		if tl.InputSchema.Type != "object" {
			t.Errorf("tool %q input schema type = %q, want object", name, tl.InputSchema.Type)
		}
	}

	// bulk_unblock and delete_policy must advertise their required args.
	if req := got["bulk_unblock"].InputSchema.Required; len(req) != 1 || req[0] != "policy_ids" {
		t.Errorf("bulk_unblock required = %v, want [policy_ids]", req)
	}
	if req := got["delete_policy"].InputSchema.Required; len(req) != 1 || req[0] != "policy_id" {
		t.Errorf("delete_policy required = %v, want [policy_id]", req)
	}
}

func TestToolsListJSONRoundTrip(t *testing.T) {
	// tools/list must round-trip through the JSON-RPC layer with the new tools.
	s := newTestServer(seededMock())
	resp := s.handleRequest(Request{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	if resp.Error != nil {
		t.Fatalf("tools/list returned error: %+v", resp.Error)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	for _, name := range []string{"get_weekly_summary", "explain_anomaly", "compare_to_last_month", "bulk_unblock", "delete_policy"} {
		if !strings.Contains(string(b), name) {
			t.Errorf("tools/list output missing %q", name)
		}
	}
}

// --- dispatch ---

func TestCallToolDispatchNewTools(t *testing.T) {
	cases := []struct {
		name string
		args string
	}{
		{"get_weekly_summary", ""},
		{"explain_anomaly", "{}"},
		{"compare_to_last_month", "{}"},
		{"bulk_unblock", `{"policy_ids":["p1"]}`},
		{"delete_policy", `{"policy_id":"p1"}`},
	}
	for _, tc := range cases {
		s := newTestServer(seededMock())
		var raw json.RawMessage
		if tc.args != "" {
			raw = json.RawMessage(tc.args)
		}
		res := s.callTool(CallToolParams{Name: tc.name, Arguments: raw})
		if len(res.Content) == 0 {
			t.Errorf("%s: empty content", tc.name)
			continue
		}
		if strings.HasPrefix(res.Content[0].Text, "Unknown tool") {
			t.Errorf("%s: not dispatched (got unknown-tool)", tc.name)
		}
		if res.IsError {
			t.Errorf("%s: unexpected error result: %s", tc.name, res.Content[0].Text)
		}
	}
}

func TestCallToolCallEndToEnd(t *testing.T) {
	// Exercise the full tools/call -> callTool path for a new tool.
	s := newTestServer(seededMock())
	params, _ := json.Marshal(CallToolParams{Name: "get_weekly_summary"})
	resp := s.handleRequest(Request{JSONRPC: "2.0", ID: 7, Method: "tools/call", Params: params})
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}
	result, ok := resp.Result.(CallToolResult)
	if !ok {
		t.Fatalf("result type = %T, want CallToolResult", resp.Result)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Content[0].Text)
	}
}

// --- formatting ---

func TestGetWeeklySummaryFormatting(t *testing.T) {
	s := newTestServer(seededMock())
	res := s.toolGetWeeklySummary()
	txt := res.Content[0].Text

	for _, want := range []string{
		"Weekly Security Summary",
		"10000 DNS queries handled",
		"2000 blocked",
		"Block rate: 20.0%",
		"grade A",
		"5 policies (4 active)",
		"3 blocklists covering 120000 domains",
		"Most-blocked domain in recent logs: ads.example.com (2 hits)",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("weekly summary missing %q\n---\n%s", want, txt)
		}
	}
}

func TestExplainAnomalyNoAnomaly(t *testing.T) {
	s := newTestServer(seededMock())
	res := s.toolExplainAnomaly(json.RawMessage(`{}`))
	if !strings.Contains(res.Content[0].Text, "No anomalies detected") {
		t.Errorf("expected no-anomaly message, got: %s", res.Content[0].Text)
	}
}

func TestExplainAnomalyHighErrorRate(t *testing.T) {
	m := seededMock()
	m.metrics.Queries.ErrorRate = 0.12 // 12%
	s := newTestServer(m)
	res := s.toolExplainAnomaly(json.RawMessage(`{}`))
	txt := res.Content[0].Text
	if !strings.Contains(txt, "Elevated DNS error rate") || !strings.Contains(txt, "12.00%") {
		t.Errorf("expected elevated error rate finding, got: %s", txt)
	}
}

func TestExplainAnomalyDegradedGrade(t *testing.T) {
	m := seededMock()
	m.metrics.Grade = "D"
	m.metrics.LatencyMs.P99 = 900
	s := newTestServer(m)
	res := s.toolExplainAnomaly(json.RawMessage(`{}`))
	if !strings.Contains(res.Content[0].Text, "Degraded latency") {
		t.Errorf("expected degraded latency finding, got: %s", res.Content[0].Text)
	}
}

func TestExplainAnomalyBlockRateSpike(t *testing.T) {
	s := newTestServer(seededMock()) // current block rate 20%
	res := s.toolExplainAnomaly(json.RawMessage(`{"baseline_block_rate":2.0}`))
	if !strings.Contains(res.Content[0].Text, "Block-rate spike") {
		t.Errorf("expected block-rate spike finding, got: %s", res.Content[0].Text)
	}

	// Small movement under the threshold must not be flagged.
	res2 := s.toolExplainAnomaly(json.RawMessage(`{"baseline_block_rate":18.0}`))
	if strings.Contains(res2.Content[0].Text, "Block-rate spike") {
		t.Errorf("did not expect spike for 2-point move, got: %s", res2.Content[0].Text)
	}
}

func TestExplainAnomalyTopClient(t *testing.T) {
	m := seededMock()
	m.logs = []api.QueryLog{
		{Domain: "a.com", ClientIP: "10.0.0.9", Action: "ALLOW"},
		{Domain: "b.com", ClientIP: "10.0.0.9", Action: "ALLOW"},
		{Domain: "c.com", ClientIP: "10.0.0.9", Action: "ALLOW"},
		{Domain: "d.com", ClientIP: "10.0.0.1", Action: "ALLOW"},
	}
	s := newTestServer(m)
	res := s.toolExplainAnomaly(json.RawMessage(`{}`))
	txt := res.Content[0].Text
	if !strings.Contains(txt, "Single client dominating traffic") || !strings.Contains(txt, "10.0.0.9") {
		t.Errorf("expected top-client finding, got: %s", txt)
	}
}

func TestCompareToLastMonthWithBaseline(t *testing.T) {
	s := newTestServer(seededMock()) // current: 10000 total, 2000 blocked, 20%
	res := s.toolCompareToLastMonth(json.RawMessage(
		`{"baseline_total_queries":8000,"baseline_blocked_queries":1000,"baseline_block_rate":12.5,"label":"June"}`))
	txt := res.Content[0].Text

	for _, want := range []string{
		"Comparison vs June:",
		"Total queries: 8000 -> 10000 (up 25.0%)",
		"Blocked queries: 1000 -> 2000 (up 100.0%)",
		"Block rate: 12.5% -> 20.0% (+7.5 points)",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("compare missing %q\n---\n%s", want, txt)
		}
	}
}

func TestCompareToLastMonthNoBaseline(t *testing.T) {
	s := newTestServer(seededMock())
	res := s.toolCompareToLastMonth(json.RawMessage(`{}`))
	txt := res.Content[0].Text
	if !strings.Contains(txt, "No baseline provided") || !strings.Contains(txt, "last month") {
		t.Errorf("expected no-baseline message defaulting to 'last month', got: %s", txt)
	}
}

// --- batch / write ---

func TestBulkUnblockAllSucceed(t *testing.T) {
	m := seededMock()
	s := newTestServer(m)
	res := s.toolBulkUnblock(json.RawMessage(`{"policy_ids":["p1","p2","p3"]}`))
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Content[0].Text)
	}
	if len(m.deleted) != 3 {
		t.Errorf("deleted %d policies, want 3", len(m.deleted))
	}
	if !strings.Contains(res.Content[0].Text, "Removed 3 of 3 policies") {
		t.Errorf("unexpected summary: %s", res.Content[0].Text)
	}
}

func TestBulkUnblockPartialFailure(t *testing.T) {
	m := seededMock()
	m.deleteErr = map[string]error{"p2": errStub("boom")}
	s := newTestServer(m)
	res := s.toolBulkUnblock(json.RawMessage(`{"policy_ids":["p1","p2"]}`))
	txt := res.Content[0].Text
	if res.IsError {
		t.Errorf("partial success should not be a hard error: %s", txt)
	}
	if !strings.Contains(txt, "Removed 1 of 2 policies") || !strings.Contains(txt, "Failed: p2") {
		t.Errorf("unexpected summary: %s", txt)
	}
}

func TestBulkUnblockAllFail(t *testing.T) {
	m := seededMock()
	m.deleteErr = map[string]error{"p1": errStub("boom")}
	s := newTestServer(m)
	res := s.toolBulkUnblock(json.RawMessage(`{"policy_ids":["p1"]}`))
	if !res.IsError {
		t.Errorf("all-fail should set IsError, got: %s", res.Content[0].Text)
	}
}

func TestBulkUnblockEmpty(t *testing.T) {
	s := newTestServer(seededMock())
	res := s.toolBulkUnblock(json.RawMessage(`{"policy_ids":[]}`))
	if !res.IsError {
		t.Errorf("empty policy_ids should be an error result")
	}
}

func TestDeletePolicy(t *testing.T) {
	m := seededMock()
	s := newTestServer(m)
	res := s.toolDeletePolicy(json.RawMessage(`{"policy_id":"abc"}`))
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Content[0].Text)
	}
	if len(m.deleted) != 1 || m.deleted[0] != "abc" {
		t.Errorf("deleted = %v, want [abc]", m.deleted)
	}
	if !strings.Contains(res.Content[0].Text, "Deleted policy: abc") {
		t.Errorf("unexpected text: %s", res.Content[0].Text)
	}
}

func TestDeletePolicyMissingID(t *testing.T) {
	s := newTestServer(seededMock())
	res := s.toolDeletePolicy(json.RawMessage(`{}`))
	if !res.IsError {
		t.Errorf("missing policy_id should be an error result")
	}
}

// errStub is a tiny error value for exercising failure paths.
type errStub string

func (e errStub) Error() string { return string(e) }
