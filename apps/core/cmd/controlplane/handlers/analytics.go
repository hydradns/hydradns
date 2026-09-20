package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
)

type QueryLogEntry struct {
	ID              uint      `json:"id"`
	Domain          string    `json:"domain"`
	ClientIP        string    `json:"client_ip"`
	Action          string    `json:"action"`
	Timestamp       time.Time `json:"timestamp"`
	IsSuspicious    bool      `json:"is_suspicious"`
	ThreatScore     float64   `json:"threat_score"`
	DetectionMethod string    `json:"detection_method,omitempty"`
	ThreatReason    string    `json:"threat_reason,omitempty"`
}

type AnalyticsSummaryData struct {
	TotalQueries     uint64  `json:"total_queries"`
	BlockedQueries   uint64  `json:"blocked_queries"`
	AllowedQueries   uint64  `json:"allowed_queries"`
	BlockRatePercent float64 `json:"block_rate_percent"`
}

type ResponseAnalyticsSummary struct {
	Status string               `json:"status"`
	Data   AnalyticsSummaryData `json:"data"`
	Error  *string              `json:"error"`
}

type ResponseQueryLogList struct {
	Status string          `json:"status"`
	Data   []QueryLogEntry `json:"data"`
	Error  *string         `json:"error"`
}

// GetAnalyticsSummary handles GET /analytics/summary
func (h *APIHandler) GetAnalyticsSummary(c *gin.Context) {
	stats, err := h.Store.Statistics.ListRecent(1)
	if err != nil || len(stats) == 0 {
		c.JSON(http.StatusOK, ResponseAnalyticsSummary{
			Status: "success",
			Data:   AnalyticsSummaryData{},
		})
		return
	}

	s := stats[0]
	var blockRate float64
	if s.TotalQueries > 0 {
		blockRate = float64(s.BlockedQueries) / float64(s.TotalQueries) * 100
	}

	c.JSON(http.StatusOK, ResponseAnalyticsSummary{
		Status: "success",
		Data: AnalyticsSummaryData{
			TotalQueries:     s.TotalQueries,
			BlockedQueries:   s.BlockedQueries,
			AllowedQueries:   s.AllowedQueries,
			BlockRatePercent: blockRate,
		},
	})
}

// queryLogEntryFromModel maps a stored row to its API shape. Shared by
// GetAuditLogs (unpaginated recent feed) and GetQueryLogsPage (paginated +
// filtered) so the two endpoints can never drift on field mapping.
//
// Method (not a free function) so it can consult h.DemoMode and redact
// client_ip — see maskClientIP in common.go for why this is applied
// unconditionally rather than trusting that demo data is always synthetic.
func (h *APIHandler) queryLogEntryFromModel(q models.DNSQuery) QueryLogEntry {
	clientIP := q.ClientIP
	if h.DemoMode {
		clientIP = maskClientIP(clientIP)
	}
	return QueryLogEntry{
		ID:              q.ID,
		Domain:          q.Domain,
		ClientIP:        clientIP,
		Action:          q.Action,
		Timestamp:       q.Timestamp,
		IsSuspicious:    q.IsSuspicious,
		ThreatScore:     q.ThreatScore,
		DetectionMethod: q.DetectionMethod,
		ThreatReason:    q.ThreatReason,
	}
}

// GetAuditLogs handles GET /analytics/audits
// Returns recent DNS query logs as the audit trail.
func (h *APIHandler) GetAuditLogs(c *gin.Context) {
	queries, err := h.Store.QueryLogs.ListRecent(100)
	if err != nil {
		errMsg := "failed to fetch query logs"
		c.JSON(http.StatusInternalServerError, ResponseQueryLogList{Status: "error", Error: &errMsg})
		return
	}

	entries := make([]QueryLogEntry, 0, len(queries))
	for _, q := range queries {
		entries = append(entries, h.queryLogEntryFromModel(q))
	}

	c.JSON(http.StatusOK, ResponseQueryLogList{
		Status: "success",
		Data:   entries,
	})
}

// Query-log pagination bounds. maxPageSize is a hard upper bound
// regardless of what the client requests — dns_queries can hold up to
// ~1,000,000 rows on an SD-card install, so an unbounded page size would
// let a single request force a huge scan+serialize.
const (
	defaultQueryLogPageSize = 50
	maxQueryLogPageSize     = 200
)

// parseQueryLogFilter parses and validates GET /analytics/logs query
// params. Matches apps/ui/lib/api.ts getQueryLogs(): client, action,
// domain, suspicious, start, end, page, page_size.
func parseQueryLogFilter(c *gin.Context) (repositories.QueryLogFilter, error) {
	f := repositories.QueryLogFilter{
		ClientIP: strings.TrimSpace(c.Query("client")),
		Domain:   strings.ToLower(strings.TrimSpace(c.Query("domain"))),
	}

	if action := strings.ToLower(strings.TrimSpace(c.Query("action"))); action != "" && action != "all" {
		f.Action = action
	}
	if c.Query("suspicious") == "true" {
		f.Suspicious = true
	}
	if s := c.Query("start"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, fmt.Errorf("invalid start: %w", err)
		}
		f.Start = &t
	}
	if s := c.Query("end"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, fmt.Errorf("invalid end: %w", err)
		}
		f.End = &t
	}

	page := 1
	if s := c.Query("page"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return f, fmt.Errorf("invalid page")
		}
		page = n
	}
	f.Page = page

	pageSize := defaultQueryLogPageSize
	if s := c.Query("page_size"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return f, fmt.Errorf("invalid page_size")
		}
		pageSize = n
	}
	if pageSize > maxQueryLogPageSize {
		pageSize = maxQueryLogPageSize
	}
	f.PageSize = pageSize

	return f, nil
}

// GetQueryLogsPage handles GET /analytics/logs: server-side pagination,
// search and filtering for the Logs page. Open to every authenticated
// role (read-only included), like the other read endpoints.
func (h *APIHandler) GetQueryLogsPage(c *gin.Context) {
	filter, err := parseQueryLogFilter(c)
	if err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": errMsg})
		return
	}

	rows, err := h.Store.QueryLogs.ListPage(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to fetch query logs"})
		return
	}
	total, err := h.Store.QueryLogs.CountFiltered(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to count query logs"})
		return
	}

	items := make([]QueryLogEntry, 0, len(rows))
	for _, q := range rows {
		items = append(items, h.queryLogEntryFromModel(q))
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      filter.Page,
			"page_size": filter.PageSize,
		},
	})
}

// bypassWindow / bypassLimit bound GET /analytics/bypass: a fixed recent
// window plus a hard cap on the number of (client, target) groups
// returned, so the aggregation stays cheap regardless of table size.
const (
	bypassWindow = 24 * time.Hour
	bypassLimit  = 50
)

// BypassAttempt is one (client, target hostname) group of encrypted-DNS
// bootstrap-interception rows. See models.DetectionMethodDoHBootstrap.
type BypassAttempt struct {
	ClientIP    string    `json:"client_ip"`
	Protocol    string    `json:"protocol"`
	Target      string    `json:"target"`
	Attempts    int64     `json:"attempts"`
	LastAttempt time.Time `json:"last_attempt"`
	Blocked     bool      `json:"blocked"`
}

// GetBypassAttempts handles GET /analytics/bypass, feeding the dashboard's
// (currently opt-in, NEXT_PUBLIC_SHOW_BYPASS_PANEL-gated) bypass-attempts
// panel. Open to every authenticated role, like the other analytics reads.
func (h *APIHandler) GetBypassAttempts(c *gin.Context) {
	since := time.Now().Add(-bypassWindow)
	summary, err := h.Store.QueryLogs.BypassAttempts(since, bypassLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to fetch bypass attempts"})
		return
	}

	attempts := make([]BypassAttempt, 0, len(summary.Rows))
	for _, r := range summary.Rows {
		clientIP := r.ClientIP
		if h.DemoMode {
			clientIP = maskClientIP(clientIP)
		}
		attempts = append(attempts, BypassAttempt{
			ClientIP: clientIP,
			// The dataplane cannot currently tell DoH from DoT from DoQ:
			// all three bootstrap via a plain A/AAAA lookup of the
			// provider's hostname (see internal/dnsengine/doh_bootstrap.go),
			// and no per-hostname protocol classification exists yet.
			// "doh" is reported as the (most common in practice) default
			// rather than a fabricated per-row distinction.
			Protocol:    "doh",
			Target:      r.Target,
			Attempts:    r.Attempts,
			LastAttempt: r.LastAttempt,
			Blocked:     r.Blocked,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"total_attempts": summary.TotalAttempts,
			"unique_clients": summary.UniqueClients,
			"attempts":       attempts,
		},
	})
}
