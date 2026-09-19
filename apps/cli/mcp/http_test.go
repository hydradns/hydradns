package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hydradns/hydra-cli/api"
)

const testBearer = "s3cr3t-bearer-token"

// newMockClient wires an api.Client to a deterministic in-memory HydraDNS
// backend. This mocks the api client so tests never touch a real server.
func newMockClient(t *testing.T) *api.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"ok","data":{`+
			`"total_policies":1,"active_policies":1,"inactive_policies":0,`+
			`"list":[{"id":"p1","name":"Block Ads","action":"BLOCK",`+
			`"domains":["ads.example.com"],"priority":150,"enabled":true}]}}`)
	})
	backend := httptest.NewServer(mux)
	t.Cleanup(backend.Close)
	return api.New(backend.URL, "test-api-token")
}

// doRPC posts a JSON-RPC body to url with an optional bearer token.
func doRPC(t *testing.T, url, tok, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// The HTTP transport dispatches a JSON-RPC tool call and returns the result,
// routed through the mocked api client.
func TestHTTPTransportDispatchesToolCall(t *testing.T) {
	s := NewServer(newMockClient(t))
	ts := httptest.NewServer(s.HTTPHandler(testBearer))
	defer ts.Close()

	body := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"list_policies"}}`
	resp := doRPC(t, ts.URL, testBearer, body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", out.Error)
	}

	raw, _ := json.Marshal(out.Result)
	if !strings.Contains(string(raw), "Block Ads") {
		t.Fatalf("tool result missing expected policy output: %s", raw)
	}
}

// tools/list is dispatched over HTTP without needing the backend.
func TestHTTPTransportListsTools(t *testing.T) {
	s := NewServer(newMockClient(t))
	ts := httptest.NewServer(s.HTTPHandler(testBearer))
	defer ts.Close()

	resp := doRPC(t, ts.URL, testBearer, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "get_status") {
		t.Fatalf("tools/list missing tools: %s", b)
	}
}

// Missing or wrong bearer token is rejected with 401 before any dispatch.
func TestHTTPTransportRejectsBadToken(t *testing.T) {
	s := NewServer(newMockClient(t))
	ts := httptest.NewServer(s.HTTPHandler(testBearer))
	defer ts.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	cases := []struct {
		name string
		tok  string
	}{
		{"missing token", ""},
		{"wrong token", "not-the-token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doRPC(t, ts.URL, tc.tok, body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}

// RunHTTP refuses to start without a token.
func TestRunHTTPRequiresToken(t *testing.T) {
	s := NewServer(nil)
	if err := s.RunHTTP("127.0.0.1:0", ""); err == nil {
		t.Fatal("expected error when starting HTTP transport with empty token")
	}
}

// The stdio path is unchanged: the shared handler dispatches an identical
// request the same way over stdio streams.
func TestStdioTransportUnchanged(t *testing.T) {
	s := NewServer(newMockClient(t))

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	if err := s.serve(in, &out); err != nil {
		t.Fatalf("stdio serve: %v", err)
	}
	if !strings.Contains(out.String(), "get_status") {
		t.Fatalf("stdio output missing tools: %s", out.String())
	}
}

// Both transports route through the same handler, so a given request yields
// the same JSON-RPC result regardless of transport.
func TestTransportsShareHandler(t *testing.T) {
	s := NewServer(newMockClient(t))

	req := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call",
		Params: json.RawMessage(`{"name":"list_policies"}`)}
	want, err := json.Marshal(s.handleRequest(req))
	if err != nil {
		t.Fatalf("marshal handler result: %v", err)
	}

	ts := httptest.NewServer(s.HTTPHandler(testBearer))
	defer ts.Close()
	resp := doRPC(t, ts.URL, testBearer,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_policies"}}`)
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)

	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("http result differs from shared handler:\n http: %s\n want: %s", got, want)
	}
}
