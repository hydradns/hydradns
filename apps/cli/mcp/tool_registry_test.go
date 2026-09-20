package mcp

import "testing"

// wantClassification is the single hand-maintained source of truth for
// this test: every tool the server registers, and whether it is
// read-only. If a new tool is added to the registry without adding an
// entry here, TestToolRegistryClassification below fails (unknown tool
// found) rather than silently defaulting to some inferred value.
var wantClassification = map[string]bool{
	"get_status":            true,
	"toggle_engine":         false,
	"block_domain":          false,
	"create_policy":         false,
	"unblock_domain":        false,
	"list_policies":         true,
	"list_blocklists":       true,
	"get_query_logs":        true,
	"get_metrics":           true,
	"get_weekly_summary":    true,
	"explain_anomaly":       true, // read-only analysis tool; name has no get_/list_ prefix
	"compare_to_last_month": true, // read-only analysis tool; name has no get_/list_ prefix
	"bulk_unblock":          false,
	"delete_policy":         false,
}

func TestToolRegistryClassification(t *testing.T) {
	reg := toolRegistry()
	if len(reg) != 14 {
		t.Fatalf("toolRegistry() has %d tools, want 14", len(reg))
	}
	seen := make(map[string]bool, len(reg))
	for _, t2 := range reg {
		seen[t2.Name] = true
		want, ok := wantClassification[t2.Name]
		if !ok {
			t.Errorf("tool %q is registered but has no expected classification in this test — add one", t2.Name)
			continue
		}
		if t2.ReadOnly != want {
			t.Errorf("tool %q: ReadOnly = %v, want %v", t2.Name, t2.ReadOnly, want)
		}
	}
	for name := range wantClassification {
		if !seen[name] {
			t.Errorf("expected tool %q not found in toolRegistry()", name)
		}
	}
}

// TestToolRegistryAllClassified guards against a tool entry being built by
// hand (a toolSpec{} literal) instead of through roTool/mutTool, which is
// the only path that marks a tool as explicitly classified. A registry
// entry that skips both constructors would otherwise silently default its
// ReadOnly field to false (mutating) with no signal that nobody actually
// decided that.
func TestToolRegistryAllClassified(t *testing.T) {
	for _, t2 := range toolRegistry() {
		if !t2.classified {
			t.Errorf("tool %q was registered without going through roTool/mutTool", t2.Name)
		}
	}
}

// isReadOnly must be driven by the registry, not a name-prefix heuristic:
// these two read-only analysis tools do not start with get_/list_.
func TestIsReadOnlyUsesRegistryNotPrefix(t *testing.T) {
	for _, name := range []string{"explain_anomaly", "compare_to_last_month"} {
		if !isReadOnly(name) {
			t.Errorf("isReadOnly(%q) = false, want true (registry-classified read-only)", name)
		}
	}
}

func TestReporterCanCallAnalysisTools(t *testing.T) {
	// This is the concrete symptom of the prefix-heuristic bug: a reporter
	// (read-only role) must be able to call explain_anomaly and
	// compare_to_last_month, which only read data.
	for _, name := range []string{"explain_anomaly", "compare_to_last_month"} {
		if !RoleReporter.Allows(name) {
			t.Errorf("reporter should be allowed to call read-only analysis tool %q", name)
		}
	}
}

// serverInfo.version must reflect whatever version the caller supplied
// (cmd/mcp.go passes cmd.Version), not a hardcoded string that can drift
// from `hydra version`.
func TestInitializeReportsSuppliedVersion(t *testing.T) {
	s := NewServer(nil, "v9.9.9")
	resp := s.handleRequest(Request{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	res, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("initialize result type = %T, want InitializeResult", resp.Result)
	}
	if res.ServerInfo.Version != "v9.9.9" {
		t.Errorf("serverInfo.version = %q, want %q", res.ServerInfo.Version, "v9.9.9")
	}
}

func TestNewServer_EmptyVersionFallsBackToDefault(t *testing.T) {
	s := NewServer(nil, "")
	resp := s.handleRequest(Request{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	res := resp.Result.(InitializeResult)
	if res.ServerInfo.Version == "" {
		t.Error("serverInfo.version must never be empty")
	}
}

func TestToolsListAnnotatesAnalysisToolsReadOnly(t *testing.T) {
	s := NewServerWithRole(nil, RoleAdmin)
	byName := make(map[string]Tool)
	for _, tl := range s.tools() {
		byName[tl.Name] = tl
	}
	for _, name := range []string{"explain_anomaly", "compare_to_last_month"} {
		tl, ok := byName[name]
		if !ok {
			t.Fatalf("tool %q not registered", name)
		}
		if tl.Annotations == nil || !tl.Annotations.ReadOnlyHint || tl.Annotations.DestructiveHint || tl.Annotations.ConfirmationRequired {
			t.Errorf("tool %q should be annotated read-only (no confirmation required), got %+v", name, tl.Annotations)
		}
	}
}
