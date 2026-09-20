package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// --- status-code handling ---

func TestDo_NonJSON2xxStillDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io := "<html>not json</html>"
		w.Write([]byte(io))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.do("GET", "/whatever", nil)
	if err == nil {
		t.Fatal("expected an error for a non-JSON body")
	}
	if !strings.Contains(err.Error(), "invalid response") {
		t.Errorf("error = %v, want it to contain %q", err, "invalid response")
	}
}

func TestDo_NonSuccessStatusFieldTreatedAsError(t *testing.T) {
	// A 200 response whose envelope does not literally say "status":"success"
	// (e.g. a proxy or captive portal echoing back an unrelated JSON body)
	// must not be treated as success just because it also isn't "error".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"foo": "bar"})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.do("GET", "/whatever", nil)
	if err == nil {
		t.Fatal("expected an error when the envelope has no status:success")
	}
}

func TestDo_NonOKStatusCodeIsErrorEvenWithoutErrorField(t *testing.T) {
	// A non-2xx HTTP status must be surfaced as an error even if the JSON
	// body has no "error" field at all (relying only on the envelope was
	// the bug).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.do("GET", "/whatever", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 502")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error = %v, want it to mention the HTTP status", err)
	}
}

func TestDo_HappyPathUnaffected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"ok": true}})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	data, err := c.do("GET", "/whatever", nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if !bytes.Contains(data, []byte("ok")) {
		t.Errorf("data = %s, want it to contain ok", data)
	}
}

// --- never store an empty token ---

func TestLogin_RejectsEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"token": ""}})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.Login("hunter2"); err == nil {
		t.Fatal("expected Login to reject a success response with an empty token")
	}
}

func TestSetup_RejectsEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{}})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	if _, err := c.Setup(SetupRequest{Password: "hunter2hunter2"}); err == nil {
		t.Fatal("expected Setup to reject a success response with no token")
	}
}

func TestLogin_AcceptsNonEmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"token": "abc123"}})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	resp, err := c.Login("hunter2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.Token != "abc123" {
		t.Errorf("token = %q, want %q", resp.Token, "abc123")
	}
}

// --- redirect must not leak Authorization cross-host ---

func TestDo_RedirectDoesNotLeakAuthorizationCrossHost(t *testing.T) {
	var gotAuth string
	var sawRequest bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{}})
	}))
	defer target.Close()

	initial := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/api/v1/somewhere-else", http.StatusTemporaryRedirect)
	}))
	defer initial.Close()

	c := New(initial.URL, "s3cr3t-token")
	if _, err := c.do("GET", "/whatever", nil); err != nil {
		t.Fatalf("do: %v", err)
	}
	if !sawRequest {
		t.Fatal("the redirect target never received a request")
	}
	if gotAuth != "" {
		t.Errorf("Authorization header leaked across a redirect to a different host: %q", gotAuth)
	}
}

func TestDo_RedirectKeepsAuthorizationSameHost(t *testing.T) {
	// Sanity check: we only want to strip Authorization when the redirect
	// target is a genuinely different host, not on every redirect.
	mux := http.NewServeMux()
	var gotAuth string
	mux.HandleFunc("/api/v1/final", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{}})
	})
	mux.HandleFunc("/api/v1/whatever", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/v1/final", http.StatusTemporaryRedirect)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL, "s3cr3t-token")
	if _, err := c.do("GET", "/whatever", nil); err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotAuth != "Bearer s3cr3t-token" {
		t.Errorf("Authorization = %q, want it preserved for a same-host redirect", gotAuth)
	}
}

// --- cleartext-HTTP warning ---

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNew_WarnsOnPublicHTTPHost(t *testing.T) {
	out := captureStderr(t, func() {
		New("http://198.51.100.7:8080", "tok")
	})
	if !strings.Contains(out, "198.51.100.7") {
		t.Errorf("stderr = %q, want a warning naming the host", out)
	}
}

func TestNew_WarnsOnPublicHostname(t *testing.T) {
	out := captureStderr(t, func() {
		New("http://dns.example.com:8080", "tok")
	})
	if out == "" {
		t.Error("expected a warning for a public-looking hostname over http://")
	}
}

func TestNew_NoWarningForLoopback(t *testing.T) {
	out := captureStderr(t, func() {
		New("http://127.0.0.1:8080", "tok")
		New("http://localhost:8080", "tok")
		New("http://[::1]:8080", "tok")
	})
	if out != "" {
		t.Errorf("stderr = %q, want no warning for loopback hosts", out)
	}
}

func TestNew_NoWarningForPrivateRanges(t *testing.T) {
	out := captureStderr(t, func() {
		New("http://192.168.1.10:8080", "tok")
		New("http://10.0.0.5:8080", "tok")
		New("http://172.16.0.5:8080", "tok")
		New("http://[fd00::1]:8080", "tok")
		New("http://[fe80::1]:8080", "tok")
	})
	if out != "" {
		t.Errorf("stderr = %q, want no warning for private/link-local hosts", out)
	}
}

func TestNew_NoWarningForHTTPS(t *testing.T) {
	out := captureStderr(t, func() {
		New("https://dns.example.com", "tok")
	})
	if out != "" {
		t.Errorf("stderr = %q, want no warning when using https://", out)
	}
}
