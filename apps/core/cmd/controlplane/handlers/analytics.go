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

// Query-log pagination bounds.
//
//   - maxQueryLogPageSize is a hard upper bound regardless of what the
//     client requests — dns_queries can hold up to ~1,000,000 rows on an
//     SD-card install, so an unbounded page size would let a single
//     request force a huge scan+serialize.
//   - maxQueryLogReach caps page*page_size: OFFSET is still a linear scan
//     in SQLite even with an index on the ORDER BY column, so
//     ?page=999999 is a cheap way for any authenticated user (including a
//     public demo visitor holding the documented demo password) to force
//     a large scan on every request. 100,000 is comfortably past anything
//     a human would page to by hand.
//   - queryLogCountCap bounds CountFilteredCapped the same way, for the
//     unindexed filters (Suspicious has no index by design) where the
//     COUNT itself would otherwise be a full table scan every time the
//     page loads.
const (
	defaultQueryLogPageSize = 50
	maxQueryLogPageSize     = 200
	maxQueryLogReach        = 100000
	queryLogCountCap        = 100000
)

// parseQueryLogFilter parses and validates GET /analytics/logs query
// params. Matches apps/ui/lib/api.ts getQueryLogs(): client, action,
// domain, suspicious, start, end, page, page_size.
//
// Method (not a free function) so it can apply h.resolveClientIPFilter —
// demo mode rejects the client filter outright (see M8: an exact-match
// filter over masked-on-output-but-unmasked-in-storage rows is an IP
// recovery oracle), and anonymization hashes it to match the hashed values
// actually stored in client_ip (see M3).
func (h *APIHandler) parseQueryLogFilter(c *gin.Context) (repositories.QueryLogFilter, error) {
	f := repositories.QueryLogFilter{
		Domain: strings.ToLower(strings.TrimSpace(c.Query("domain"))),
	}

	if rawClient := strings.TrimSpace(c.Query("client")); rawClient != "" {
		clientFilter, err := h.resolveClientIPFilter(rawClient)
		if err != nil {
			return f, err
		}
		f.ClientIP = clientFilter
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

	if int64(f.Page)*int64(f.PageSize) > maxQueryLogReach {
		return f, fmt.Errorf("page %d with page_size %d exceeds the maximum reachable offset (page*page_size must be <= %d)", f.Page, f.PageSize, maxQueryLogReach)
	}

	return f, nil
}

// countQueryLogs uses the bounded-cost count (repositories.QueryLogCapper)
// when the configured QueryLogRepository implements it — true for the real
// GormQueryLogRepo — and falls back to plain CountFiltered otherwise. See
// QueryLogCapper's doc comment for why this is a type assertion rather
// than a new method on QueryLogRepository itself.
func (h *APIHandler) countQueryLogs(filter repositories.QueryLogFilter) (total int64, capped bool, err error) {
	if capper, ok := h.Store.QueryLogs.(repositories.QueryLogCapper); ok {
		return capper.CountFilteredCapped(filter, queryLogCountCap)
	}
	n, err := h.Store.QueryLogs.CountFiltered(filter)
	return n, false, err
}

// GetQueryLogsPage handles GET /analytics/logs: server-side pagination,
// search and filtering for the Logs page. Open to every authenticated
// role (read-only included), like the other read endpoints.
func (h *APIHandler) GetQueryLogsPage(c *gin.Context) {
	filter, err := h.parseQueryLogFilter(c)
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
	total, capped, err := h.countQueryLogs(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": "failed to count query logs"})
		return
	}

	items := make([]QueryLogEntry, 0, len(rows))
	for _, q := range rows {
		items = append(items, h.queryLogEntryFromModel(q))
	}

	data := gin.H{
		"items":     items,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	}
	if capped {
		// Extra field, ignored by clients that don't know about it (see
		// apps/ui/lib/types.ts's QueryLogPage — a plain TS interface with
		// no runtime schema validation on the fetch path). Lets the UI
		// render "100,000+" instead of a number that looks exact but was
		// deliberately never computed past the cap.
		data["total_capped"] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
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
