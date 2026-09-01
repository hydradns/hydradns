package mcp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hydradns/hydra-cli/api"
)

// apiClient is the subset of *api.Client the MCP server depends on. It exists
// as a seam so tools can be exercised against mocked control-plane data in
// tests without a live backend. *api.Client satisfies it.
type apiClient interface {
	GetEngineStatus() (*api.EngineStatus, error)
	GetDashboardSummary() (*api.DashboardSummary, error)
	ToggleEngine(enabled bool) error
	CreatePolicy(req map[string]interface{}) (*api.Policy, error)
	DeletePolicy(id string) error
	ListPolicies() (*api.PolicyListData, error)
	ListBlocklists() (*api.BlocklistListData, error)
	GetQueryLogs() ([]api.QueryLog, error)
	GetMetrics() (*api.Metrics, error)
}

// toolGetWeeklySummary builds a natural-language rollup from the existing
// stats, metrics, blocklist, and query-log endpoints.
func (s *Server) toolGetWeeklySummary() CallToolResult {
	summary, err := s.client.GetDashboardSummary()
	if err != nil {
		return errorResult(err)
	}
	metrics, err := s.client.GetMetrics()
	if err != nil {
		return errorResult(err)
	}
	policies, err := s.client.ListPolicies()
	if err != nil {
		return errorResult(err)
	}
	blocklists, err := s.client.ListBlocklists()
	if err != nil {
		return errorResult(err)
	}
	logs, err := s.client.GetQueryLogs()
	if err != nil {
		return errorResult(err)
	}

	var b strings.Builder
	b.WriteString("HydraDNS Weekly Security Summary\n")
	b.WriteString("================================\n")

	fmt.Fprintf(&b, "Traffic: %d DNS queries handled (%d blocked, %d allowed, %d redirected).\n",
		summary.TotalQueries, summary.BlockedQueries, summary.AllowedQueries, summary.RedirectedQueries)
	fmt.Fprintf(&b, "Block rate: %.1f%% of queries were filtered.\n", summary.BlockRatePercent)
	fmt.Fprintf(&b, "Performance: p50=%dms, p95=%dms, p99=%dms latency (grade %s), error rate %.2f%%.\n",
		metrics.LatencyMs.P50, metrics.LatencyMs.P95, metrics.LatencyMs.P99,
		metrics.Grade, metrics.Queries.ErrorRate*100)
	fmt.Fprintf(&b, "Protection: %d policies (%d active), %d blocklists covering %d domains.\n",
		policies.TotalPolicies, policies.ActivePolicies,
		blocklists.TotalBlocklists, blocklists.TotalDomains)

	if domain, count := topBlockedDomain(logs); count > 0 {
		fmt.Fprintf(&b, "Most-blocked domain in recent logs: %s (%d hits).\n", domain, count)
	} else {
		b.WriteString("No blocked queries in the recent log sample.\n")
	}

	return textResult(strings.TrimRight(b.String(), "\n"))
}

type anomalyArgs struct {
	BaselineBlockRate    *float64 `json:"baseline_block_rate"`
	MaxErrorRatePercent  *float64 `json:"max_error_rate_percent"`
	BlockRateJumpPercent *float64 `json:"block_rate_jump_percent"`
}

// toolExplainAnomaly diffs current activity against thresholds (and an optional
// baseline block rate) and describes anything unusual in plain language.
func (s *Server) toolExplainAnomaly(argsRaw json.RawMessage) CallToolResult {
	var p anomalyArgs
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &p); err != nil {
			return errorResult(fmt.Errorf("invalid arguments: %w", err))
		}
	}

	maxErrorRate := 5.0
	if p.MaxErrorRatePercent != nil {
		maxErrorRate = *p.MaxErrorRatePercent
	}
	blockRateJump := 15.0
	if p.BlockRateJumpPercent != nil {
		blockRateJump = *p.BlockRateJumpPercent
	}

	summary, err := s.client.GetDashboardSummary()
	if err != nil {
		return errorResult(err)
	}
	metrics, err := s.client.GetMetrics()
	if err != nil {
		return errorResult(err)
	}
	logs, err := s.client.GetQueryLogs()
	if err != nil {
		return errorResult(err)
	}

	var findings []string

	errorRate := metrics.Queries.ErrorRate * 100
	if errorRate > maxErrorRate {
		findings = append(findings, fmt.Sprintf(
			"Elevated DNS error rate: %.2f%% (threshold %.2f%%). Resolver reachability or upstream failures are likely.",
			errorRate, maxErrorRate))
	}

	if isDegradedGrade(metrics.Grade) {
		findings = append(findings, fmt.Sprintf(
			"Degraded latency: grade %s with p99=%dms. Queries are resolving slower than usual.",
			metrics.Grade, metrics.LatencyMs.P99))
	}

	if p.BaselineBlockRate != nil {
		delta := summary.BlockRatePercent - *p.BaselineBlockRate
		if delta > blockRateJump {
			findings = append(findings, fmt.Sprintf(
				"Block-rate spike: now %.1f%%, up %.1f points from the baseline of %.1f%%. A new blocklist or a burst of malicious lookups may be responsible.",
				summary.BlockRatePercent, delta, *p.BaselineBlockRate))
		}
	}

	if ip, count, share := topClient(logs); share > 0.5 {
		findings = append(findings, fmt.Sprintf(
			"Single client dominating traffic: %s accounts for %d of %d logged queries (%.0f%%).",
			ip, count, len(logs), share*100))
	}

	if len(findings) == 0 {
		return textResult("No anomalies detected. Error rate, latency, block rate, and client distribution are all within normal ranges.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Detected %d anomal%s:\n", len(findings), plural(len(findings), "y", "ies"))
	for i, f := range findings {
		fmt.Fprintf(&b, "%d. %s\n", i+1, f)
	}
	return textResult(strings.TrimRight(b.String(), "\n"))
}

type compareArgs struct {
	BaselineTotalQueries   *uint64  `json:"baseline_total_queries"`
	BaselineBlockedQueries *uint64  `json:"baseline_blocked_queries"`
	BaselineBlockRate      *float64 `json:"baseline_block_rate"`
	Label                  string   `json:"label"`
}

// toolCompareToLastMonth diffs current traffic against caller-supplied baseline
// figures and narrates the change.
func (s *Server) toolCompareToLastMonth(argsRaw json.RawMessage) CallToolResult {
	var p compareArgs
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &p); err != nil {
			return errorResult(fmt.Errorf("invalid arguments: %w", err))
		}
	}
	label := p.Label
	if label == "" {
		label = "last month"
	}

	summary, err := s.client.GetDashboardSummary()
	if err != nil {
		return errorResult(err)
	}

	if p.BaselineTotalQueries == nil && p.BaselineBlockedQueries == nil && p.BaselineBlockRate == nil {
		return textResult(fmt.Sprintf(
			"No baseline provided, so there is nothing to compare against. Current activity: %d queries, %d blocked, block rate %.1f%%. "+
				"Pass baseline_total_queries, baseline_blocked_queries, and/or baseline_block_rate to compare against %s.",
			summary.TotalQueries, summary.BlockedQueries, summary.BlockRatePercent, label))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Comparison vs %s:\n", label)

	if p.BaselineTotalQueries != nil {
		fmt.Fprintf(&b, "Total queries: %d -> %d (%s).\n",
			*p.BaselineTotalQueries, summary.TotalQueries,
			describeChangeUint(*p.BaselineTotalQueries, summary.TotalQueries))
	}
	if p.BaselineBlockedQueries != nil {
		fmt.Fprintf(&b, "Blocked queries: %d -> %d (%s).\n",
			*p.BaselineBlockedQueries, summary.BlockedQueries,
			describeChangeUint(*p.BaselineBlockedQueries, summary.BlockedQueries))
	}
	if p.BaselineBlockRate != nil {
		delta := summary.BlockRatePercent - *p.BaselineBlockRate
		fmt.Fprintf(&b, "Block rate: %.1f%% -> %.1f%% (%+.1f points).\n",
			*p.BaselineBlockRate, summary.BlockRatePercent, delta)
	}

	return textResult(strings.TrimRight(b.String(), "\n"))
}

type bulkUnblockArgs struct {
	PolicyIDs []string `json:"policy_ids"`
}

// toolBulkUnblock deletes several policies in one call, mirroring the
// DELETE /policies/:id endpoint used by unblock_domain.
func (s *Server) toolBulkUnblock(argsRaw json.RawMessage) CallToolResult {
	var p bulkUnblockArgs
	if err := json.Unmarshal(argsRaw, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if len(p.PolicyIDs) == 0 {
		return errorResult(fmt.Errorf("no policy_ids provided"))
	}

	var removed []string
	var failed []string
	for _, id := range p.PolicyIDs {
		if err := s.client.DeletePolicy(id); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%s)", id, err.Error()))
			continue
		}
		removed = append(removed, id)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Removed %d of %d policies.\n", len(removed), len(p.PolicyIDs))
	if len(removed) > 0 {
		fmt.Fprintf(&b, "Removed: %s\n", strings.Join(removed, ", "))
	}
	if len(failed) > 0 {
		fmt.Fprintf(&b, "Failed: %s\n", strings.Join(failed, ", "))
	}

	res := textResult(strings.TrimRight(b.String(), "\n"))
	if len(removed) == 0 {
		res.IsError = true
	}
	return res
}

type deletePolicyArgs struct {
	PolicyID string `json:"policy_id"`
}

// toolDeletePolicy deletes a single policy by ID (BLOCK, ALLOW, or REDIRECT),
// mirroring the DELETE /policies/:id endpoint.
func (s *Server) toolDeletePolicy(argsRaw json.RawMessage) CallToolResult {
	var p deletePolicyArgs
	if err := json.Unmarshal(argsRaw, &p); err != nil {
		return errorResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if p.PolicyID == "" {
		return errorResult(fmt.Errorf("policy_id is required"))
	}
	if err := s.client.DeletePolicy(p.PolicyID); err != nil {
		return errorResult(err)
	}
	return textResult(fmt.Sprintf("Deleted policy: %s", p.PolicyID))
}

// --- helpers ---

// topBlockedDomain returns the most frequently blocked domain in the log sample
// and its hit count. count is 0 when there are no blocked entries.
func topBlockedDomain(logs []api.QueryLog) (string, int) {
	counts := map[string]int{}
	for _, l := range logs {
		if strings.EqualFold(l.Action, "BLOCK") || strings.EqualFold(l.Action, "BLOCKED") {
			counts[l.Domain]++
		}
	}
	return topByCount(counts)
}

// topClient returns the busiest client IP in the log sample, its query count,
// and its share of the sample (0..1). share is 0 when there are no logs.
func topClient(logs []api.QueryLog) (string, int, float64) {
	if len(logs) == 0 {
		return "", 0, 0
	}
	counts := map[string]int{}
	for _, l := range logs {
		counts[l.ClientIP]++
	}
	ip, count := topByCount(counts)
	return ip, count, float64(count) / float64(len(logs))
}

// topByCount returns the key with the highest count, breaking ties by key name
// for deterministic output. Returns ("", 0) for an empty map.
func topByCount(counts map[string]int) (string, int) {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bestKey, bestCount := "", 0
	for _, k := range keys {
		if counts[k] > bestCount {
			bestKey, bestCount = k, counts[k]
		}
	}
	return bestKey, bestCount
}

func isDegradedGrade(grade string) bool {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "C", "D", "F":
		return true
	default:
		return false
	}
}

func describeChangeUint(from, to uint64) string {
	if from == to {
		return "no change"
	}
	if from == 0 {
		return "up from zero"
	}
	pct := (float64(to) - float64(from)) / float64(from) * 100
	direction := "up"
	if to < from {
		direction = "down"
	}
	return fmt.Sprintf("%s %.1f%%", direction, absFloat(pct))
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
