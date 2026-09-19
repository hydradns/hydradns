// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildCORSRouter wires CORS() into a minimal router the same way main.go
// does (r.Use(middlewares.CORS()) ahead of the route handlers), so tests
// exercise the real middleware construction rather than calling internal
// helpers directly.
func buildCORSRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/api/v1/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func corsRequest(r http.Handler, method, origin, host string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/test", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if host != "" {
		req.Host = host
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", "GET")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestCORS_ExplicitAllowlistHit(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "http://localhost:3000", "localhost:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:3000", got)
	}
}

func TestCORS_ExplicitAllowlistFromEnv(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://dashboard.example.com")
	t.Setenv("CORS_ALLOW_SAME_HOST", "false")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "https://dashboard.example.com", "192.168.1.53:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestCORS_ExplicitMiss(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	// Unrelated origin and host: not in the allowlist, and hostnames don't
	// match each other, so the same-host fallback can't save it either.
	rec := corsRequest(r, http.MethodGet, "http://malicious.example:1234", "192.168.1.53:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestCORS_SameHostIPv4Allowed(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	// Dashboard opened at http://192.168.1.53:3000 calling the API at
	// http://192.168.1.53:8080 — not in the static allowlist, must come
	// from the same-host fallback.
	rec := corsRequest(r, http.MethodGet, "http://192.168.1.53:3000", "192.168.1.53:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://192.168.1.53:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://192.168.1.53:3000", got)
	}
}

func TestCORS_SameHostIPv6Allowed(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "http://[fe80::1]:3000", "[fe80::1]:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestCORS_SameHostLocalhostAllowed(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	// Port 4000 keeps this off the static localhost:3000 allowlist entry,
	// isolating the same-host fallback for "localhost" specifically.
	rec := corsRequest(r, http.MethodGet, "http://localhost:4000", "localhost:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestCORS_DifferentIPDenied(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "http://192.168.1.53:3000", "10.0.0.5:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestCORS_NamedHostSameHostDenied_Rebinding(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	// Origin host and Host header agree ("evil.example"), simulating a
	// successful DNS rebind, but a named host is never covered by the
	// automatic rule.
	rec := corsRequest(r, http.MethodGet, "http://evil.example:3000", "evil.example:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403 (rebinding case must stay denied)", rec.Code)
	}
}

func TestCORS_SameHostDisabledViaEnv(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "false")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "http://192.168.1.53:3000", "192.168.1.53:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403 (same-host rule disabled)", rec.Code)
	}
}

func TestCORS_PreflightSameHostAllowed(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodOptions, "http://192.168.1.53:3000", "192.168.1.53:8080")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight got %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://192.168.1.53:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://192.168.1.53:3000", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods missing from preflight response")
	}
}

func TestCORS_NullOriginDenied(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "null", "192.168.1.53:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestCORS_MalformedOriginDenied(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	// No scheme at all — fails the http/https scheme check.
	rec := corsRequest(r, http.MethodGet, "192.168.1.53:3000", "192.168.1.53:8080")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestCORS_NoOriginHeaderPassesThrough(t *testing.T) {
	// Non-browser clients (curl, the CLI) never send an Origin header. The
	// CORS middleware is a no-op for them by design (gin-contrib/cors
	// applyCors returns immediately when Origin is empty); auth still
	// gates the request via the separate Auth middleware, not CORS.
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("CORS_ALLOW_SAME_HOST", "")
	r := buildCORSRouter()

	rec := corsRequest(r, http.MethodGet, "", "192.168.1.53:8080")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

// --- Unit tests for the helpers directly ---

func TestHostnameOnly(t *testing.T) {
	cases := []struct{ in, want string }{
		{"192.168.1.53:8080", "192.168.1.53"},
		{"192.168.1.53", "192.168.1.53"},
		{"localhost:8080", "localhost"},
		{"[fe80::1]:8080", "fe80::1"},
		{"[fe80::1]", "fe80::1"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := hostnameOnly(tc.in); got != tc.want {
			t.Errorf("hostnameOnly(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
