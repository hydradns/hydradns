// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"strings"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/logger"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"gorm.io/gorm"
)

// Interface (clean, mockable)
type QueryLogRepository interface {
	Save(query *models.DNSQuery) error
	SaveBatch(queries []*models.DNSQuery) error
	ListRecent(limit int) ([]models.DNSQuery, error)
	DeleteOlderThan(cutoff time.Time) (int64, error)
	EnforceRowCap(maxRows int64) (int64, error)
	Count() (int64, error)

	// ListPage and CountFiltered back GET /analytics/logs: server-side
	// pagination + filtering over a table that can hold up to ~1M rows on
	// an SD card. See applyFilter for the indexing rationale.
	ListPage(filter QueryLogFilter) ([]models.DNSQuery, error)
	CountFiltered(filter QueryLogFilter) (int64, error)

	// BypassAttempts aggregates DoH/DoT/DoQ bootstrap-interception rows
	// (models.DetectionMethodDoHBootstrap) for the /analytics/bypass panel.
	BypassAttempts(since time.Time, limit int) (BypassSummary, error)
}

// QueryLogCapper is an optional capability of a QueryLogRepository: a
// bounded-cost count for GET /analytics/logs (see GormQueryLogRepo.
// CountFilteredCapped). Deliberately NOT part of QueryLogRepository
// itself: that interface is implemented by hand-written fakes elsewhere
// in the module (internal/dnsengine's tests, outside this change's scope)
// that have no reason to grow a new method just because the control
// plane's logs endpoint needs a cheaper count. Callers (see
// handlers.APIHandler.countQueryLogs) type-assert for this interface and
// fall back to plain CountFiltered when a repository doesn't implement
// it.
type QueryLogCapper interface {
	CountFilteredCapped(filter QueryLogFilter, capAt int64) (total int64, capped bool, err error)
}

// QueryLogFilter narrows ListPage/CountFiltered. Zero-value fields are
// ignored. Page is 1-indexed; PageSize is expected to already be clamped
// by the caller (see handlers.parseQueryLogFilter).
type QueryLogFilter struct {
	Domain     string
	ClientIP   string
	Action     string
	Suspicious bool
	Start      *time.Time
	End        *time.Time
	Page       int
	PageSize   int
}

// BypassSummary is the aggregated result behind GET /analytics/bypass.
type BypassSummary struct {
	TotalAttempts int64
	UniqueClients int64
	Rows          []BypassAttemptRow
}

// BypassAttemptRow is one (client, target hostname) group.
type BypassAttemptRow struct {
	ClientIP    string
	Target      string
	Attempts    int64
	LastAttempt time.Time
	// Blocked is true if at least one attempt in the group was answered
	// with a block action. Given the dataplane's current design (Step 0
	// of ProcessDNSQuery unconditionally intercepts every bootstrap
	// hostname lookup with NXDOMAIN), this is always true today. Kept as
	// a real aggregate rather than a hardcoded constant so it stays
	// correct if that ever changes.
	Blocked bool
}

// Implementation
type GormQueryLogRepo struct {
	db *gorm.DB
}

func NewGormQueryLogRepo(db *gorm.DB) *GormQueryLogRepo {
	return &GormQueryLogRepo{db: db}
}

func (r *GormQueryLogRepo) Save(query *models.DNSQuery) error {
	query.Timestamp = time.Now()
	logger.Log.Debug("Saving DNS query log")
	logger.Log.Debug("query", query)
	return r.db.Create(query).Error
}

// SaveBatch inserts many query logs in a single multi-row transaction.
// Used by the async log writer so a high query rate collapses into few
// DB round-trips instead of one INSERT (and one fsync) per query.
func (r *GormQueryLogRepo) SaveBatch(queries []*models.DNSQuery) error {
	if len(queries) == 0 {
		return nil
	}
	now := time.Now()
	for _, q := range queries {
		if q.Timestamp.IsZero() {
			q.Timestamp = now
		}
	}
	return r.db.CreateInBatches(queries, 200).Error
}

func (r *GormQueryLogRepo) ListRecent(limit int) ([]models.DNSQuery, error) {
	var queries []models.DNSQuery
	err := r.db.Order("timestamp desc").Limit(limit).Find(&queries).Error
	return queries, err
}

// Count returns the number of rows in the query log table.
func (r *GormQueryLogRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.DNSQuery{}).Count(&n).Error
	return n, err
}

// DeleteOlderThan removes query logs with a timestamp before cutoff and
// returns the number of rows deleted. Timestamp is indexed, so this is a
// ranged delete, not a full scan.
func (r *GormQueryLogRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	res := r.db.Where("timestamp < ?", cutoff).Delete(&models.DNSQuery{})
	return res.RowsAffected, res.Error
}

// EnforceRowCap keeps at most maxRows of the newest query logs, deleting
// the oldest beyond that. This is disk insurance: a sustained query rate
// can exceed any time-window estimate, and the Pi's SD card must not fill.
// IDs are monotonic, so "newest" == "highest id". Returns rows deleted.
func (r *GormQueryLogRepo) EnforceRowCap(maxRows int64) (int64, error) {
	if maxRows <= 0 {
		return 0, nil
	}
	// Find the id of the maxRows-th newest row; everything with a smaller
	// id is surplus. OFFSET maxRows-1 lands on the last row we keep.
	var threshold uint
	err := r.db.Model(&models.DNSQuery{}).
		Order("id desc").
		Offset(int(maxRows-1)).
		Limit(1).
		Pluck("id", &threshold).Error
	if err != nil {
		return 0, err
	}
	if threshold == 0 {
		// Fewer than maxRows rows present; nothing to trim.
		return 0, nil
	}
	res := r.db.Where("id < ?", threshold).Delete(&models.DNSQuery{})
	return res.RowsAffected, res.Error
}

// sqliteTimeLayouts are the datetime text formats glebarez/sqlite (and
// SQLite generally) may store a time.Time column as, tried in order.
// GORM parses these automatically when scanning into a recognized model
// field; parseSQLiteTime does the same for ad-hoc aggregate columns (see
// BypassAttempts).
var sqliteTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
}

func parseSQLiteTime(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range sqliteTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// escapeLike escapes SQL LIKE metacharacters (and the escape character
// itself) in user-supplied search text, so a domain/IP filter containing
// "%" or "_" is matched literally instead of as a wildcard. Combined with
// ESCAPE '\' on every LIKE clause below.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// applyFilter builds the WHERE clauses shared by ListPage and
// CountFiltered. All user-supplied values are passed as bound parameters
// (GORM placeholders), never concatenated into the query string.
//
// Indexing notes (dns_queries can hold up to ~1,000,000 rows on an SD
// card, and this query runs on every logs-page filter keystroke):
//   - Domain uses a prefix match ("x%"), not a leading-wildcard substring
//     search, specifically so it can use the existing index on `domain`
//     (a leading "%x%" can never use a B-tree index and would force a
//     full table scan on every keystroke).
//   - ClientIP matches only the exact value or the exact value followed
//     by a literal ":" (legacy port-suffixed rows, see engine.go's
//     logQuery). Both branches are index-usable (equality + non-leading-
//     wildcard prefix) and neither is a loose prefix: "192.168.1.5" does
//     not match "192.168.1.50:1111".
//   - Action has an index (added alongside this feature) since the
//     Blocked/Allowed pills filter by it on every page load/click.
//   - Start/End use the existing `timestamp` index (range scan).
//   - Suspicious (a rarely-used checkbox, not typed per keystroke) has no
//     index; a full scan here is an accepted, occasional cost.
func (r *GormQueryLogRepo) applyFilter(q *gorm.DB, f QueryLogFilter) *gorm.DB {
	if f.Domain != "" {
		q = q.Where("domain LIKE ? ESCAPE '\\'", escapeLike(f.Domain)+"%")
	}
	if f.ClientIP != "" {
		esc := escapeLike(f.ClientIP)
		q = q.Where("client_ip = ? OR client_ip LIKE ? ESCAPE '\\'", f.ClientIP, esc+":%")
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Suspicious {
		q = q.Where("is_suspicious = ?", true)
	}
	if f.Start != nil {
		q = q.Where("timestamp >= ?", *f.Start)
	}
	if f.End != nil {
		q = q.Where("timestamp <= ?", *f.End)
	}
	return q
}

// ListPage returns one page of query logs matching filter, newest first
// (id desc breaks ties on identical timestamps so pagination is stable
// even when many rows share a timestamp).
func (r *GormQueryLogRepo) ListPage(f QueryLogFilter) ([]models.DNSQuery, error) {
	q := r.applyFilter(r.db.Model(&models.DNSQuery{}), f).
		Order("timestamp desc, id desc")
	if f.PageSize > 0 {
		q = q.Limit(f.PageSize)
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	if offset := (page - 1) * f.PageSize; offset > 0 {
		q = q.Offset(offset)
	}
	var rows []models.DNSQuery
	err := q.Find(&rows).Error
	return rows, err
}

// CountFiltered returns the total row count matching filter (ignoring
// Page/PageSize), so the UI can render pagination without a second
// unfiltered round-trip.
func (r *GormQueryLogRepo) CountFiltered(f QueryLogFilter) (int64, error) {
	var n int64
	err := r.applyFilter(r.db.Model(&models.DNSQuery{}), f).Count(&n).Error
	return n, err
}

// CountFilteredCapped is CountFiltered but bounds the work SQLite actually
// does: rather than "SELECT COUNT(*) FROM dns_queries WHERE ..." (a full
// scan for an unindexed filter like Suspicious, over a table that can hold
// ~1M rows), it counts rows from a subquery capped at capAt+1 matches
// ("SELECT COUNT(*) FROM (SELECT 1 FROM dns_queries WHERE ... LIMIT
// capAt+1)"). If the subquery hits its limit, the true total is unknown
// (could be anything >= capAt) but doesn't matter for pagination purposes:
// the UI is told the total is capAt and that it's capped, which is
// enough to render "100,000+" instead of computing an exact count nobody
// can page through anyway (see the reachable-offset cap in
// handlers.parseQueryLogFilter). capAt<=0 disables capping.
func (r *GormQueryLogRepo) CountFilteredCapped(f QueryLogFilter, capAt int64) (total int64, capped bool, err error) {
	if capAt <= 0 {
		n, err := r.CountFiltered(f)
		return n, false, err
	}

	sub := r.applyFilter(r.db.Model(&models.DNSQuery{}), f).
		Select("1").
		Limit(int(capAt + 1))

	var n int64
	if err := r.db.Table("(?) as capped_rows", sub).Count(&n).Error; err != nil {
		return 0, false, err
	}
	if n > capAt {
		return capAt, true, nil
	}
	return n, false, nil
}

// bypassFiltered is the shared WHERE clause for the bypass-attempts
// aggregation: rows the dataplane tagged as DoH/DoT/DoQ bootstrap
// interceptions, within the given window. Timestamp is indexed, so this
// stays a bounded range scan even at 1M rows.
func (r *GormQueryLogRepo) bypassFiltered(since time.Time) *gorm.DB {
	return r.db.Model(&models.DNSQuery{}).
		Where("detection_method = ? AND timestamp >= ?", models.DetectionMethodDoHBootstrap, since)
}

// BypassAttempts aggregates bootstrap-interception rows by (client, target
// hostname) within [since, now]. Bounded by limit (top groups by attempt
// count) so this stays cheap regardless of table size.
func (r *GormQueryLogRepo) BypassAttempts(since time.Time, limit int) (BypassSummary, error) {
	var summary BypassSummary

	if err := r.bypassFiltered(since).Count(&summary.TotalAttempts).Error; err != nil {
		return summary, err
	}
	if summary.TotalAttempts == 0 {
		return summary, nil
	}
	if err := r.bypassFiltered(since).Distinct("client_ip").Count(&summary.UniqueClients).Error; err != nil {
		return summary, err
	}

	// LastAttempt is scanned as a string, not time.Time: GORM's automatic
	// datetime parsing only applies when scanning into a recognized model
	// field (e.g. models.DNSQuery.Timestamp); an ad-hoc aggregate column
	// like MAX(timestamp) comes back as the raw SQLite TEXT value via
	// database/sql's generic scan path, which does not know how to
	// convert a string into time.Time. Parsed below with parseSQLiteTime.
	type row struct {
		ClientIP    string
		Target      string
		Attempts    int64
		LastAttempt string
		BlockedN    int64
	}
	var rows []row
	err := r.bypassFiltered(since).
		Select("client_ip AS client_ip, domain AS target, COUNT(*) AS attempts, MAX(timestamp) AS last_attempt, SUM(CASE WHEN action = 'block' THEN 1 ELSE 0 END) AS blocked_n").
		Group("client_ip, domain").
		Order("attempts DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return summary, err
	}

	summary.Rows = make([]BypassAttemptRow, 0, len(rows))
	for _, rr := range rows {
		lastAttempt, perr := parseSQLiteTime(rr.LastAttempt)
		if perr != nil {
			// Shipping a row with the zero time (0001-01-01T00:00:00Z)
			// looks like real, if very old, data to a caller: the UI would
			// render "25000 years ago" with no indication anything is
			// wrong. Drop the row instead;
			// TotalAttempts/UniqueClients above were already computed from
			// separate queries and are unaffected, so the aggregate counts
			// stay accurate even though this one group's row is withheld.
			logger.Log.Warnf("bypass attempts: dropping row for %s/%s, unparseable last_attempt %q: %v", rr.ClientIP, rr.Target, rr.LastAttempt, perr)
			continue
		}
		summary.Rows = append(summary.Rows, BypassAttemptRow{
			ClientIP:    rr.ClientIP,
			Target:      rr.Target,
			Attempts:    rr.Attempts,
			LastAttempt: lastAttempt,
			Blocked:     rr.BlockedN > 0,
		})
	}
	return summary, nil
}
