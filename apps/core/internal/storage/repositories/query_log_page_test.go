// SPDX-License-Identifier: Apache-2.0
package repositories

import (
	"testing"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
)

// --- ListPage / CountFiltered ---

func TestQueryLog_ListPage_NoFilters_OrderedNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	base := time.Now().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		db.Create(&models.DNSQuery{
			Domain: "example.com", ClientIP: "10.0.0.1", Action: "allow",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		})
	}

	rows, err := repo.ListPage(QueryLogFilter{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !rows[0].Timestamp.After(rows[1].Timestamp) || !rows[1].Timestamp.After(rows[2].Timestamp) {
		t.Errorf("expected newest-first ordering, got %v", rows)
	}

	total, err := repo.CountFiltered(QueryLogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestQueryLog_ListPage_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 5; i++ {
		db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})
	}

	page1, err := repo.ListPage(QueryLogFilter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2 rows, got %d", len(page1))
	}

	page3, err := repo.ListPage(QueryLogFilter{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1 row (5 total, page size 2), got %d", len(page3))
	}
}

func TestQueryLog_Filter_Domain_PrefixMatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "ads.example.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "example.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "other.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})

	rows, err := repo.ListPage(QueryLogFilter{Domain: "example", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Domain != "example.com" {
		t.Errorf("expected only the domain starting with %q, got %+v", "example", rows)
	}

	total, _ := repo.CountFiltered(QueryLogFilter{Domain: "example"})
	if total != 1 {
		t.Errorf("expected count 1, got %d", total)
	}
}

func TestQueryLog_Filter_Domain_EscapesLikeWildcards(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "a_b.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "axb.com", ClientIP: "10.0.0.1", Action: "allow", Timestamp: time.Now()})

	// "a_b" must not act as a SQL LIKE wildcard and match "axb.com" too.
	rows, err := repo.ListPage(QueryLogFilter{Domain: "a_b", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Domain != "a_b.com" {
		t.Errorf("expected LIKE wildcards in user input to be escaped, got %+v", rows)
	}
}

func TestQueryLog_Filter_ClientIP_ExactMatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "192.168.1.5", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "192.168.1.50", Action: "allow", Timestamp: time.Now()})

	rows, err := repo.ListPage(QueryLogFilter{ClientIP: "192.168.1.5", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ClientIP != "192.168.1.5" {
		t.Errorf("expected exact match only (not a prefix match hitting .50), got %+v", rows)
	}
}

func TestQueryLog_Filter_ClientIP_MatchesLegacyPortSuffixPrecisely(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	// Legacy rows stored before the port-stripping fix.
	db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "192.168.1.5:54321", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "192.168.1.50:1111", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "c.com", ClientIP: "192.168.1.5", Action: "allow", Timestamp: time.Now()})

	rows, err := repo.ListPage(QueryLogFilter{ClientIP: "192.168.1.5", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (bare ip + ip:port for 192.168.1.5, NOT 192.168.1.50:1111), got %d: %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.ClientIP != "192.168.1.5" && r.ClientIP != "192.168.1.5:54321" {
			t.Errorf("unexpected row matched: %+v", r)
		}
	}
}

func TestQueryLog_Filter_Action(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "1.1.1.1", Action: "block", Timestamp: time.Now()})

	rows, err := repo.ListPage(QueryLogFilter{Action: "block", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Action != "block" {
		t.Errorf("expected only blocked rows, got %+v", rows)
	}
}

func TestQueryLog_Filter_Suspicious(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", IsSuspicious: false, Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "1.1.1.1", Action: "allow", IsSuspicious: true, Timestamp: time.Now()})

	rows, err := repo.ListPage(QueryLogFilter{Suspicious: true, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].IsSuspicious {
		t.Errorf("expected only suspicious rows, got %+v", rows)
	}
}

func TestQueryLog_Filter_TimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	now := time.Now()
	db.Create(&models.DNSQuery{Domain: "old.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: now.Add(-2 * time.Hour)})
	db.Create(&models.DNSQuery{Domain: "mid.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: now.Add(-1 * time.Hour)})
	db.Create(&models.DNSQuery{Domain: "new.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: now})

	start := now.Add(-90 * time.Minute)
	end := now.Add(-30 * time.Minute)
	rows, err := repo.ListPage(QueryLogFilter{Start: &start, End: &end, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Domain != "mid.com" {
		t.Errorf("expected only mid.com within the window, got %+v", rows)
	}
}

// --- CountFilteredCapped bounds the COUNT query itself ---

func TestQueryLog_CountFilteredCapped_BelowCapReturnsExactTotal(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 5; i++ {
		db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	}

	total, capped, err := repo.CountFilteredCapped(QueryLogFilter{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || capped {
		t.Errorf("expected total=5 capped=false, got total=%d capped=%v", total, capped)
	}
}

func TestQueryLog_CountFilteredCapped_AboveCapReturnsCappedTotal(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 10; i++ {
		db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	}

	total, capped, err := repo.CountFilteredCapped(QueryLogFilter{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !capped {
		t.Error("expected capped=true when actual rows exceed the cap")
	}
	if total != 3 {
		t.Errorf("expected the reported total to be clamped to the cap (3), got %d", total)
	}
}

func TestQueryLog_CountFilteredCapped_ExactlyAtCapIsNotCapped(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 3; i++ {
		db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	}

	total, capped, err := repo.CountFilteredCapped(QueryLogFilter{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if capped {
		t.Error("expected capped=false when the actual count exactly equals the cap")
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
}

func TestQueryLog_CountFilteredCapped_RespectsFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "block", Timestamp: time.Now()})
	db.Create(&models.DNSQuery{Domain: "b.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})

	total, capped, err := repo.CountFilteredCapped(QueryLogFilter{Action: "block"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || capped {
		t.Errorf("expected total=1 capped=false, got total=%d capped=%v", total, capped)
	}
}

func TestQueryLog_CountFilteredCapped_ZeroCapMeansUnbounded(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 5; i++ {
		db.Create(&models.DNSQuery{Domain: "a.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: time.Now()})
	}

	total, capped, err := repo.CountFilteredCapped(QueryLogFilter{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || capped {
		t.Errorf("expected total=5 capped=false with capAt=0 (unbounded), got total=%d capped=%v", total, capped)
	}
}

func TestQueryLog_ListPage_StableOrderingOnTies(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	ts := time.Now().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		db.Create(&models.DNSQuery{Domain: "same.com", ClientIP: "1.1.1.1", Action: "allow", Timestamp: ts})
	}

	rows, err := repo.ListPage(QueryLogFilter{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// Same timestamp: tie-break must be id desc (stable, deterministic).
	if !(rows[0].ID > rows[1].ID && rows[1].ID > rows[2].ID) {
		t.Errorf("expected id-desc tie-break ordering, got ids %d,%d,%d", rows[0].ID, rows[1].ID, rows[2].ID)
	}
}
