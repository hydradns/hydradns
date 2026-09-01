package mcp

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxRequestBytes caps the size of an inbound JSON-RPC request body on the
// HTTP transport to avoid unbounded reads.
const maxRequestBytes = 1 << 20 // 1 MiB

// RunHTTP serves the MCP protocol as JSON-RPC 2.0 over HTTP, an optional
// transport alongside stdio for driving a managed fleet remotely.
//
// Every request must carry "Authorization: Bearer <token>"; requests with a
// missing or incorrect token are rejected with 401 before any dispatch. A
// non-empty token is mandatory. This transport carries management traffic
// (engine/policy control), never DNS query data.
func (s *Server) RunHTTP(addr, token string) error {
	if token == "" {
		return fmt.Errorf("mcp http transport requires a bearer token")
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.HTTPHandler(token),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// HTTPHandler returns an http.Handler that authenticates each request with the
// bearer token and dispatches the JSON-RPC request through the same
// handleRequest path used by the stdio transport. It is exported so it can be
// driven directly by tests (httptest) without binding a socket.
func (s *Server) HTTPHandler(token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Authenticate before doing any work.
		if !authorized(r, token) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			// Mirror the stdio transport: parse failures come back as a
			// JSON-RPC error object (HTTP 200), not a transport error.
			writeJSONRPC(w, Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &Error{Code: -32700, Message: "Parse error"},
			})
			return
		}

		// Notifications (no ID expected) get no response body.
		if strings.HasPrefix(req.Method, "notifications/") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		writeJSONRPC(w, s.handleRequest(req))
	})
	return mux
}

// writeJSONRPC writes a JSON-RPC Response with HTTP 200. Application-level
// failures (bad method, invalid params, tool errors) are conveyed inside the
// JSON-RPC envelope, per the protocol; only transport/auth problems use HTTP
// status codes.
func writeJSONRPC(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(resp)
	_, _ = w.Write(b)
}

// authorized reports whether the request carries the expected bearer token.
// The comparison is constant-time and an empty configured or supplied token is
// always rejected.
func authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	got := bearerToken(r)
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header, or returns "" if absent/malformed.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
