package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	warnIfInsecure(baseURL)
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		http: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: stripAuthorizationCrossHost,
		},
	}
}

// stripAuthorizationCrossHost is a Client.CheckRedirect policy. Go's
// default redirect handling already drops Authorization/Cookie headers
// when the redirect target's *hostname* differs from the original
// request's, but it only compares hostnames (net/http
// shouldCopyHeaderOnRedirect / isDomainOrSubdomain), not host:port. A
// redirect to the same hostname on a different port still forwards the
// bearer token, and for an admin API token that's the more realistic
// local attack: a compromised or spoofed service on another port of the
// same box. This compares the full host:port instead and strips
// Authorization on any mismatch. It also preserves the standard
// 10-redirect cap, since setting CheckRedirect overrides Go's default.
func stripAuthorizationCrossHost(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if len(via) > 0 && req.URL.Host != via[0].URL.Host {
		req.Header.Del("Authorization")
	}
	return nil
}

// warnIfInsecure prints a one-line, stderr-only warning when baseURL is
// plain http:// and the host is not loopback, RFC1918, ULA, or
// link-local, meaning it looks like a public address or a public-looking
// name. HydraDNS has no TLS story yet (see CLAUDE.md "Known Incomplete
// Features"), and plain http:// on a LAN is the normal, expected way to
// reach the box; this only fires for the case that actually leaks a
// password or bearer token over the public internet in cleartext. It
// never writes to stdout, so it is safe under `hydra mcp` stdio mode
// where stdout is the JSON-RPC channel.
func warnIfInsecure(baseURL string) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "http" {
		return
	}
	host := u.Hostname()
	if host == "" || hostLooksPrivate(host) {
		return
	}
	fmt.Fprintf(os.Stderr,
		"warning: %s uses plain http:// to a non-local address (%s); credentials will be sent in cleartext over the network\n",
		baseURL, host)
}

// hostLooksPrivate reports whether host is loopback, RFC1918/ULA, or
// link-local. A public IP, or a hostname that would need a network
// round trip to resolve, is treated as public-looking so the warning
// fires rather than staying silent.
func hostLooksPrivate(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// API response envelope
type Response struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *string         `json:"error"`
}

func (c *Client) do(method, path string, body interface{}) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+"/api/v1"+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	var r Response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	// Check the HTTP status in addition to the envelope: a proxy or
	// captive portal in front of the API can return a non-2xx status with
	// a body that doesn't match our envelope at all (no "error" field),
	// or a 2xx with valid-looking JSON that simply isn't ours (no
	// "status":"success"). Relying on r.Status != "error" alone treated
	// both of those as success.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || r.Status != "success" {
		msg := "unknown error"
		switch {
		case r.Error != nil && *r.Error != "":
			msg = *r.Error
		case resp.StatusCode < 200 || resp.StatusCode >= 300:
			msg = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error: %s", msg)
	}
	return r.Data, nil
}

// --- Types ---

type DashboardSummary struct {
	TotalQueries      uint64  `json:"total_queries"`
	BlockedQueries    uint64  `json:"blocked_queries"`
	AllowedQueries    uint64  `json:"allowed_queries"`
	RedirectedQueries uint64  `json:"redirected_queries"`
	BlockRatePercent  float64 `json:"block_rate_percent"`
}

type EngineStatus struct {
	Enabled          bool   `json:"enabled"`
	AcceptingQueries bool   `json:"accepting_queries"`
	LastError        string `json:"last_error"`
}

type Metrics struct {
	WindowSeconds int `json:"window_seconds"`
	Queries       struct {
		Total     int     `json:"total"`
		Errors    int     `json:"errors"`
		ErrorRate float64 `json:"error_rate"`
	} `json:"queries"`
	LatencyMs struct {
		P50 uint64 `json:"p50"`
		P95 uint64 `json:"p95"`
		P99 uint64 `json:"p99"`
	} `json:"latency_ms"`
	Grade string `json:"grade"`
}

type Policy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Action      string   `json:"action"`
	RedirectIP  string   `json:"redirect_ip,omitempty"`
	Domains     []string `json:"domains"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
}

type PolicyListData struct {
	TotalPolicies    int      `json:"total_policies"`
	ActivePolicies   int      `json:"active_policies"`
	InactivePolicies int      `json:"inactive_policies"`
	List             []Policy `json:"list"`
}

type Blocklist struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Format       string `json:"format"`
	Category     string `json:"category"`
	DomainsCount int64  `json:"domains_count"`
	Enabled      bool   `json:"enabled"`
}

type BlocklistListData struct {
	TotalBlocklists int         `json:"total_blocklists"`
	TotalDomains    int64       `json:"total_domains"`
	ActiveLists     []Blocklist `json:"active_lists"`
}

type QueryLog struct {
	ID        uint   `json:"id"`
	Domain    string `json:"domain"`
	ClientIP  string `json:"client_ip"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

type Resolver struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
}

// --- Methods ---

func (c *Client) GetDashboardSummary() (*DashboardSummary, error) {
	data, err := c.do("GET", "/dashboard/summary", nil)
	if err != nil {
		return nil, err
	}
	var s DashboardSummary
	return &s, json.Unmarshal(data, &s)
}

func (c *Client) GetEngineStatus() (*EngineStatus, error) {
	data, err := c.do("GET", "/dns/engine", nil)
	if err != nil {
		return nil, err
	}
	var s EngineStatus
	return &s, json.Unmarshal(data, &s)
}

func (c *Client) ToggleEngine(enabled bool) error {
	_, err := c.do("POST", "/dns/engine", map[string]bool{"enabled": enabled})
	return err
}

func (c *Client) GetMetrics() (*Metrics, error) {
	data, err := c.do("GET", "/dns/metrics", nil)
	if err != nil {
		return nil, err
	}
	var m Metrics
	return &m, json.Unmarshal(data, &m)
}

func (c *Client) GetResolvers() ([]Resolver, error) {
	data, err := c.do("GET", "/dns/resolvers", nil)
	if err != nil {
		return nil, err
	}
	var r []Resolver
	return r, json.Unmarshal(data, &r)
}

func (c *Client) ListPolicies() (*PolicyListData, error) {
	data, err := c.do("GET", "/policies", nil)
	if err != nil {
		return nil, err
	}
	var p PolicyListData
	return &p, json.Unmarshal(data, &p)
}

func (c *Client) CreatePolicy(req map[string]interface{}) (*Policy, error) {
	data, err := c.do("POST", "/policies", req)
	if err != nil {
		return nil, err
	}
	var p Policy
	return &p, json.Unmarshal(data, &p)
}

func (c *Client) DeletePolicy(id string) error {
	_, err := c.do("DELETE", "/policies/"+id, nil)
	return err
}

func (c *Client) ListBlocklists() (*BlocklistListData, error) {
	data, err := c.do("GET", "/blocklists", nil)
	if err != nil {
		return nil, err
	}
	var b BlocklistListData
	return &b, json.Unmarshal(data, &b)
}

func (c *Client) CreateBlocklist(req map[string]interface{}) (*Blocklist, error) {
	data, err := c.do("POST", "/blocklists", req)
	if err != nil {
		return nil, err
	}
	var b Blocklist
	return &b, json.Unmarshal(data, &b)
}

func (c *Client) DeleteBlocklist(id string) error {
	_, err := c.do("DELETE", "/blocklists/"+id, nil)
	return err
}

type AuthStatus struct {
	SetupComplete bool `json:"setup_complete"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// SetupBlocklistRequest mirrors setupBlocklistRequest on the server
// (apps/core cmd/controlplane/handlers/auth.go). The setup wizard in
// the dashboard covers blocklist selection, so this isn't exposed via
// CLI flags, but it's kept here so the wire contract lives in one place.
type SetupBlocklistRequest struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Format string `json:"format,omitempty"`
}

// SetupRequest mirrors setupRequest on the server. Email is optional;
// the server defaults to admin@hydradns.local when omitted.
type SetupRequest struct {
	Email      string                  `json:"email,omitempty"`
	Password   string                  `json:"password"`
	Blocklists []SetupBlocklistRequest `json:"blocklists,omitempty"`
}

type SetupResponse struct {
	Token    string   `json:"token"`
	Warnings []string `json:"warnings,omitempty"`
}

func (c *Client) GetAuthStatus() (*AuthStatus, error) {
	data, err := c.do("GET", "/auth/status", nil)
	if err != nil {
		return nil, err
	}
	var s AuthStatus
	return &s, json.Unmarshal(data, &s)
}

// Setup calls POST /auth/setup to create the first admin user and mint
// its long-lived token. Only succeeds if no users exist yet (the server
// returns a 409 "setup already completed" error otherwise).
func (c *Client) Setup(req SetupRequest) (*SetupResponse, error) {
	data, err := c.do("POST", "/auth/setup", req)
	if err != nil {
		return nil, err
	}
	var r SetupResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	// A "success" envelope with no token is not success: never let the
	// caller persist an empty token (see saveToken in cmd/token_store.go).
	if r.Token == "" {
		return nil, fmt.Errorf("server returned a success response with no token")
	}
	return &r, nil
}

func (c *Client) Login(password string) (*LoginResponse, error) {
	data, err := c.do("POST", "/auth/login", map[string]string{"password": password})
	if err != nil {
		return nil, err
	}
	var r LoginResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Token == "" {
		return nil, fmt.Errorf("server returned a success response with no token")
	}
	return &r, nil
}

func (c *Client) GetQueryLogs() ([]QueryLog, error) {
	data, err := c.do("GET", "/analytics/audits", nil)
	if err != nil {
		return nil, err
	}
	var logs []QueryLog
	return logs, json.Unmarshal(data, &logs)
}
