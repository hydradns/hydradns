package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
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
	if r.Status == "error" {
		msg := "unknown error"
		if r.Error != nil {
			msg = *r.Error
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

func (c *Client) GetAuthStatus() (*AuthStatus, error) {
	data, err := c.do("GET", "/auth/status", nil)
	if err != nil {
		return nil, err
	}
	var s AuthStatus
	return &s, json.Unmarshal(data, &s)
}

func (c *Client) Login(password string) (*LoginResponse, error) {
	data, err := c.do("POST", "/auth/login", map[string]string{"password": password})
	if err != nil {
		return nil, err
	}
	var r LoginResponse
	return &r, json.Unmarshal(data, &r)
}

func (c *Client) GetQueryLogs() ([]QueryLog, error) {
	data, err := c.do("GET", "/analytics/audits", nil)
	if err != nil {
		return nil, err
	}
	var logs []QueryLog
	return logs, json.Unmarshal(data, &logs)
}
