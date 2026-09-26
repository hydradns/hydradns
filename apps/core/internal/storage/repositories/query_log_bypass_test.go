// SPDX-License-Identifier: Apache-2.0
package repositories

import (
	"testing"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
)

// TestQueryLog_BypassAttempts_DropsRowWithUnparseableTimestamp verifies a
// row whose last_attempt text doesn't match any of the recognized SQLite
// datetime layouts is dropped rather than shipped with the Go zero time
// (0001-01-01T00:00:00Z), which looks like real (if absurdly old) data to
// any caller. It must be dropped from the per-group Rows instead, while
// the separately-computed aggregate counts (TotalAttempts, UniqueClients)
// stay accurate.
func TestQueryLog_BypassAttempts_DropsRowWithUnparseableTimestamp(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	// A well-formed row, written the normal way.
	db.Create(&models.DNSQuery{
		Domain: "dns.google", ClientIP: "192.168.1.5", Action: "block",
		DetectionMethod: models.DetectionMethodDoHBootstrap, Timestamp: time.Now(),
	})

	// A row whose timestamp text is not one of sqliteTimeLayouts, inserted
	// via raw SQL so the on-disk text is exactly controlled (GORM would
	// otherwise always write a parseable format).
	if err := db.Exec(
		`INSERT INTO dns_queries (domain, client_ip, action, detection_method, timestamp) VALUES (?, ?, ?, ?, ?)`,
		"cloudflare-dns.com", "192.168.1.9", "block", models.DetectionMethodDoHBootstrap, "not-a-timestamp",
	).Error; err != nil {
		t.Fatalf("seed malformed row: %v", err)
	}

	summary, err := repo.BypassAttempts(time.Now().Add(-time.Hour), 50)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalAttempts != 2 {
		t.Errorf("expected TotalAttempts to still count both rows (computed independently of the parse), got %d", summary.TotalAttempts)
	}
	if len(summary.Rows) != 1 {
		t.Fatalf("expected the unparseable-timestamp group to be dropped from Rows, got %d rows: %+v", len(summary.Rows), summary.Rows)
	}
	if summary.Rows[0].ClientIP != "192.168.1.5" {
		t.Errorf("expected the well-formed row to survive, got %+v", summary.Rows[0])
	}
	if summary.Rows[0].LastAttempt.IsZero() {
		t.Error("expected the surviving row to have a real, non-zero LastAttempt")
	}
}
