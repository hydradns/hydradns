package mcp

import (
	"fmt"
	"strings"
)

// Role represents an MCP permission scope. It is resolved from the MCP_ROLE
// environment variable (or an explicit value) and determines which tools a
// caller may invoke.
type Role string

const (
	// RoleAdmin may call every tool. This is the default when MCP_ROLE is unset
	// or empty, so existing behaviour is unchanged unless a role is configured.
	RoleAdmin Role = "admin"
	// RoleOperator may call read-only tools and mutating tools, but may not
	// toggle the DNS engine on or off.
	RoleOperator Role = "operator"
	// RoleReporter may call read-only tools only.
	RoleReporter Role = "reporter"
)

// codePermissionDenied is the JSON-RPC error code returned when a tool call is
// rejected because the active role does not permit it. It falls within the
// JSON-RPC 2.0 implementation-defined server-error range (-32000 to -32099).
const codePermissionDenied = -32003

// resolveRole maps a raw MCP_ROLE value to a Role. An empty/unset value
// defaults to admin (no restriction) to preserve backward compatibility. Any
// unrecognised value safe-defaults to the most restrictive role (reporter,
// read-only only) rather than silently granting elevated access.
func resolveRole(raw string) Role {
	switch Role(strings.ToLower(strings.TrimSpace(raw))) {
	case "":
		return RoleAdmin
	case RoleAdmin:
		return RoleAdmin
	case RoleOperator:
		return RoleOperator
	case RoleReporter:
		return RoleReporter
	default:
		return RoleReporter
	}
}

// isReadOnly reports whether a tool only reads state. Read-only tools are named
// with a get_ or list_ prefix and are always permitted for every role.
func isReadOnly(toolName string) bool {
	return strings.HasPrefix(toolName, "get_") || strings.HasPrefix(toolName, "list_")
}

// Allows reports whether the role may invoke the named tool.
//
//   - Read-only tools (get_*, list_*) are always allowed.
//   - admin may call any tool.
//   - operator may call any mutating tool except toggle_engine.
//   - reporter (and any unknown/safe-defaulted role) may call read-only tools only.
func (r Role) Allows(toolName string) bool {
	if isReadOnly(toolName) {
		return true
	}
	switch r {
	case RoleAdmin:
		return true
	case RoleOperator:
		return toolName != "toggle_engine"
	default: // reporter and the safe-default for unknown roles
		return false
	}
}

// permissionError builds the structured JSON-RPC error returned when a role is
// not permitted to call a tool.
func permissionError(role Role, toolName string) *Error {
	return &Error{
		Code:    codePermissionDenied,
		Message: fmt.Sprintf("permission denied: role %q is not permitted to call tool %q", role, toolName),
		Data: map[string]any{
			"role":   string(role),
			"tool":   toolName,
			"reason": "role_not_permitted",
		},
	}
}
