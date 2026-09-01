package mcp

import (
	"encoding/json"
	"testing"
)

func TestResolveRole(t *testing.T) {
	cases := []struct {
		raw  string
		want Role
	}{
		{"", RoleAdmin},                // unset/empty -> default admin (backward compatible)
		{"admin", RoleAdmin},           //
		{"ADMIN", RoleAdmin},           // case-insensitive
		{"  operator  ", RoleOperator}, // trimmed
		{"reporter", RoleReporter},     //
		{"wizard", RoleReporter},       // unknown -> safe default (most restrictive)
		{"root", RoleReporter},         // unknown -> safe default
	}
	for _, c := range cases {
		if got := resolveRole(c.raw); got != c.want {
			t.Errorf("resolveRole(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestRoleAllows(t *testing.T) {
	readOnly := []string{"get_status", "get_query_logs", "get_metrics", "list_policies", "list_blocklists"}
	mutating := []string{"toggle_engine", "block_domain", "unblock_domain", "create_policy", "delete_policy", "bulk_import"}

	// Read-only tools are always allowed, for every role including an
	// unknown role that has been safe-defaulted.
	for _, role := range []Role{RoleAdmin, RoleOperator, RoleReporter, resolveRole("wizard")} {
		for _, tool := range readOnly {
			if !role.Allows(tool) {
				t.Errorf("role %q should allow read-only tool %q", role, tool)
			}
		}
	}

	// admin may call everything.
	for _, tool := range mutating {
		if !RoleAdmin.Allows(tool) {
			t.Errorf("admin should allow mutating tool %q", tool)
		}
	}

	// operator may call mutating tools EXCEPT toggle_engine.
	if RoleOperator.Allows("toggle_engine") {
		t.Errorf("operator must not be allowed to call toggle_engine")
	}
	for _, tool := range []string{"block_domain", "unblock_domain", "create_policy", "delete_policy", "bulk_import"} {
		if !RoleOperator.Allows(tool) {
			t.Errorf("operator should allow mutating tool %q", tool)
		}
	}

	// reporter (and unknown safe-default) may call NO mutating tool.
	for _, role := range []Role{RoleReporter, resolveRole("wizard")} {
		for _, tool := range mutating {
			if role.Allows(tool) {
				t.Errorf("role %q must not be allowed to call mutating tool %q", role, tool)
			}
		}
	}
}

// callTools drives handleRequest for a tools/call and returns the response.
// The client is nil: denied calls must return before ever touching it, so the
// test stays deterministic and network-free.
func callToolRequest(s *Server, name string) Response {
	args, _ := json.Marshal(CallToolParams{Name: name})
	return s.handleRequest(Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  args,
	})
}

func TestToolsCallGateDenies(t *testing.T) {
	cases := []struct {
		role Role
		tool string
	}{
		{RoleReporter, "block_domain"},           // reporter blocked from a mutating tool
		{RoleReporter, "create_policy"},          //
		{RoleOperator, "toggle_engine"},          // operator blocked from toggle_engine
		{resolveRole("wizard"), "toggle_engine"}, // unknown role safe-defaults to deny
	}
	for _, c := range cases {
		s := NewServerWithRole(nil, c.role)
		resp := callToolRequest(s, c.tool)
		if resp.Error == nil {
			t.Fatalf("role %q calling %q: expected JSON-RPC error, got none", c.role, c.tool)
		}
		if resp.Error.Code != codePermissionDenied {
			t.Errorf("role %q calling %q: error code = %d, want %d", c.role, c.tool, resp.Error.Code, codePermissionDenied)
		}
		if resp.Result != nil {
			t.Errorf("role %q calling %q: result must be nil on denial, got %v", c.role, c.tool, resp.Result)
		}
	}
}

func TestToolsListAnnotations(t *testing.T) {
	s := NewServerWithRole(nil, RoleAdmin)
	resp := s.handleRequest(Request{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	res, ok := resp.Result.(ToolsListResult)
	if !ok {
		t.Fatalf("tools/list result type = %T, want ToolsListResult", resp.Result)
	}

	byName := make(map[string]Tool, len(res.Tools))
	for _, tl := range res.Tools {
		byName[tl.Name] = tl
	}

	ro, ok := byName["get_status"]
	if !ok || ro.Annotations == nil || !ro.Annotations.ReadOnlyHint || ro.Annotations.DestructiveHint {
		t.Errorf("get_status should be annotated read-only, got %+v", ro.Annotations)
	}

	mut, ok := byName["toggle_engine"]
	if !ok || mut.Annotations == nil || !mut.Annotations.DestructiveHint || !mut.Annotations.ConfirmationRequired {
		t.Errorf("toggle_engine should be annotated destructive + confirmation-required, got %+v", mut.Annotations)
	}
}
