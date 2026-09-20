package repositories

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	db.AutoMigrate(
		&models.BlocklistSource{},
		&models.BlocklistSnapshot{},
		&models.BlocklistEntry{},
		&models.DNSQuery{},
		&models.Statistics{},
		&models.SystemState{},
		&models.Policy{},
	)
	return db
}

// --- Blocklist Repository ---

func TestBlocklistRepo_IsBlocked(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	// Insert a blocklist entry
	db.Create(&models.BlocklistEntry{Domain: "blocked.com", SourceID: "test"})

	blocked, err := repo.IsBlocked("blocked.com")
	if err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Error("expected blocked.com to be blocked")
	}

	blocked, err = repo.IsBlocked("allowed.com")
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Error("expected allowed.com to not be blocked")
	}
}

func TestBlocklistRepo_IsBlocked_Normalization(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	db.Create(&models.BlocklistEntry{Domain: "example.com", SourceID: "test"})

	tests := []struct {
		input string
		want  bool
	}{
		{"EXAMPLE.COM", true},
		{"Example.Com.", true},
		{"example.com.", true},
		{"example.com", true},
		{"other.com", false},
	}
	for _, tt := range tests {
		got, err := repo.IsBlocked(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("IsBlocked(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestBlocklistRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	db.Create(&models.BlocklistEntry{Domain: "a.com", SourceID: "test"})
	db.Create(&models.BlocklistEntry{Domain: "b.com", SourceID: "test"})

	domains, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}
}

func TestBlocklistRepo_SaveSnapshotWithEntries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := models.BlocklistSource{
		ID: "test-src", Name: "Test", URL: "http://example.com", Format: "hosts",
		Enabled: true, CreatedAt: time.Now(),
	}
	db.Create(&src)

	entries := []models.BlocklistEntry{
		{Domain: "a.com", SourceID: "test-src", Category: "ads"},
		{Domain: "b.com", SourceID: "test-src", Category: "ads"},
		{Domain: "c.com", SourceID: "test-src", Category: "ads"},
	}

	snap, err := repo.SaveSnapshotWithEntries(src, "abc123", entries)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Size != 3 {
		t.Errorf("expected snapshot size 3, got %d", snap.Size)
	}
	if snap.Checksum != "abc123" {
		t.Errorf("expected checksum abc123, got %s", snap.Checksum)
	}

	// Verify entries are persisted
	blocked, _ := repo.IsBlocked("a.com")
	if !blocked {
		t.Error("expected a.com to be blocked after snapshot save")
	}
}

// A source's entries must be the *current* snapshot's contents, not the
// union of every snapshot ever ingested. A domain the upstream list dropped
// must stop being blocked once the next fetch lands.
func TestBlocklistRepo_SaveSnapshotWithEntries_ReplacesPriorEntries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := models.BlocklistSource{ID: "src1", Name: "Test", URL: "http://example.com/hosts", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	db.Create(&src)

	first := []models.BlocklistEntry{
		{Domain: "a.com", SourceID: "src1"},
		{Domain: "b.com", SourceID: "src1"},
	}
	if _, err := repo.SaveSnapshotWithEntries(src, "checksum-1", first); err != nil {
		t.Fatal(err)
	}

	second := []models.BlocklistEntry{
		{Domain: "b.com", SourceID: "src1"},
		{Domain: "c.com", SourceID: "src1"},
	}
	if _, err := repo.SaveSnapshotWithEntries(src, "checksum-2", second); err != nil {
		t.Fatal(err)
	}

	domains, err := repo.GetAllEnabled()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"b.com": true, "c.com": true}
	if len(domains) != len(want) {
		t.Fatalf("GetAllEnabled() = %v, want exactly %v", domains, want)
	}
	for _, d := range domains {
		if !want[d] {
			t.Errorf("unexpected stale domain %q survived the second snapshot", d)
		}
	}
	blockedA, _ := repo.IsBlocked("a.com")
	if blockedA {
		t.Error("a.com was dropped from the upstream list and must no longer be blocked")
	}

	count, err := repo.CountEntriesBySource("src1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("domains_count = %d, want 2 (current snapshot only)", count)
	}
}

// A fetch that finds the content unchanged (same checksum as the
// source's last successful ingest) must be a no-op: it must not delete and
// re-insert the identical entries.
func TestBlocklistRepo_SaveSnapshotWithEntries_UnchangedChecksumLeavesEntriesAlone(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := models.BlocklistSource{ID: "src1", Name: "Test", URL: "http://example.com/hosts", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	db.Create(&src)

	entries := []models.BlocklistEntry{{Domain: "a.com", SourceID: "src1"}, {Domain: "b.com", SourceID: "src1"}}
	firstSnap, err := repo.SaveSnapshotWithEntries(src, "same-checksum", entries)
	if err != nil {
		t.Fatal(err)
	}

	// Re-fetch: reload the source as the caller would (LastHash now set),
	// and save again with the identical checksum.
	fresh, err := repo.GetSource("src1")
	if err != nil {
		t.Fatal(err)
	}
	secondSnap, err := repo.SaveSnapshotWithEntries(*fresh, "same-checksum", entries)
	if err != nil {
		t.Fatal(err)
	}
	if secondSnap.ID != firstSnap.ID {
		t.Errorf("unchanged checksum created a new snapshot row (id %d != %d); expected a no-op", secondSnap.ID, firstSnap.ID)
	}

	count, err := repo.CountEntriesBySource("src1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("unchanged fetch touched entries: domains_count = %d, want 2", count)
	}
}

// Editing a source's URL (or the upstream content changing entirely)
// followed by a fetch must leave only the new list's domains: none of the
// old URL's entries survive.
func TestBlocklistRepo_SaveSnapshotWithEntries_URLEditThenFetchLeavesOnlyNewList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := models.BlocklistSource{ID: "src1", Name: "Test", URL: "http://old.example.com/hosts", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	db.Create(&src)

	oldEntries := []models.BlocklistEntry{{Domain: "old-a.com", SourceID: "src1"}, {Domain: "old-b.com", SourceID: "src1"}}
	if _, err := repo.SaveSnapshotWithEntries(src, "old-checksum", oldEntries); err != nil {
		t.Fatal(err)
	}

	// Operator edits the URL (mirrors handlers.UpdateBlocklist's UpdateSourceFields call).
	src.URL = "http://new.example.com/hosts"
	if err := repo.UpdateSourceFields(&src); err != nil {
		t.Fatal(err)
	}

	// Next fetch (of the new URL) completes with entirely different content.
	newEntries := []models.BlocklistEntry{{Domain: "new-a.com", SourceID: "src1"}}
	if _, err := repo.SaveSnapshotWithEntries(src, "new-checksum", newEntries); err != nil {
		t.Fatal(err)
	}

	domains, err := repo.GetAllEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "new-a.com" {
		t.Fatalf("GetAllEnabled() = %v, want [new-a.com] only (old URL's entries must be gone)", domains)
	}
	blockedOld, _ := repo.IsBlocked("old-a.com")
	if blockedOld {
		t.Error("old-a.com from the pre-edit URL is still blocked after the URL edit's fetch completed")
	}
}

// Retention: snapshot *metadata* rows are capped per source so the table
// doesn't grow without bound across years of refreshes, while still keeping
// some history. The exact cap is an implementation choice (see
// snapshotRetentionPerSource); this test only asserts it is enforced and
// that the most recent snapshot is always kept.
func TestBlocklistRepo_SaveSnapshotWithEntries_PrunesOldSnapshotMetadata(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := models.BlocklistSource{ID: "src1", Name: "Test", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	db.Create(&src)

	var lastSnap models.BlocklistSnapshot
	for i := 0; i < snapshotRetentionPerSource+5; i++ {
		fresh, err := repo.GetSource("src1")
		if err != nil {
			t.Fatal(err)
		}
		entries := []models.BlocklistEntry{{Domain: "a.com", SourceID: "src1"}}
		snap, err := repo.SaveSnapshotWithEntries(*fresh, fmt.Sprintf("checksum-%d", i), entries)
		if err != nil {
			t.Fatal(err)
		}
		lastSnap = snap
	}

	var count int64
	if err := db.Model(&models.BlocklistSnapshot{}).Where("source_id = ?", "src1").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != snapshotRetentionPerSource {
		t.Errorf("snapshot metadata rows = %d, want %d (retention cap)", count, snapshotRetentionPerSource)
	}

	var stillThere models.BlocklistSnapshot
	if err := db.First(&stillThere, "id = ?", lastSnap.ID).Error; err != nil {
		t.Errorf("most recent snapshot (id %d) was pruned: %v", lastSnap.ID, err)
	}
}

// DeleteSource's and the replace-on-ingest DELETE both filter by
// source_id; without an index both are full table scans over
// blocklist_entries, which can hold millions of rows on a Pi.
func TestBlocklistRepo_BlocklistEntriesSourceIDIsIndexed(t *testing.T) {
	db := setupTestDB(t)

	var indexNames []string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'blocklist_entries'").
		Scan(&indexNames).Error; err != nil {
		t.Fatal(err)
	}

	found := false
	for _, name := range indexNames {
		var col string
		if err := db.Raw("SELECT name FROM pragma_index_info(?) LIMIT 1", name).Scan(&col).Error; err != nil {
			t.Fatal(err)
		}
		if col == "source_id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no index on blocklist_entries.source_id; found indexes: %v", indexNames)
	}
}

func TestBlocklistRepo_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	domains, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(domains))
	}
}

// --- Query Log Repository ---

func TestQueryLogRepo_SaveAndListRecent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	repo.Save(&models.DNSQuery{Domain: "first.com", ClientIP: "1.2.3.4", Action: "allow"})
	repo.Save(&models.DNSQuery{Domain: "second.com", ClientIP: "1.2.3.4", Action: "block"})

	queries, err := repo.ListRecent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 {
		t.Errorf("expected 2 queries, got %d", len(queries))
	}
	// Most recent first
	if queries[0].Domain != "second.com" {
		t.Errorf("expected most recent first, got %q", queries[0].Domain)
	}
}

func TestQueryLogRepo_ListRecent_Limit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormQueryLogRepo(db)

	for i := 0; i < 5; i++ {
		repo.Save(&models.DNSQuery{Domain: "test.com", ClientIP: "1.2.3.4", Action: "allow"})
	}

	queries, err := repo.ListRecent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 3 {
		t.Errorf("expected 3 queries with limit, got %d", len(queries))
	}
}

// --- Statistics Repository ---

func TestStatisticsRepo_IncrementCounter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormStatisticsRepo(db)

	repo.IncrementCounter("allow")
	repo.IncrementCounter("allow")
	repo.IncrementCounter("block")
	repo.IncrementCounter("redirect")

	var stats models.Statistics
	db.First(&stats, 1)

	if stats.TotalQueries != 4 {
		t.Errorf("expected 4 total, got %d", stats.TotalQueries)
	}
	if stats.AllowedQueries != 2 {
		t.Errorf("expected 2 allowed, got %d", stats.AllowedQueries)
	}
	if stats.BlockedQueries != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.BlockedQueries)
	}
	if stats.RedirectedQueries != 1 {
		t.Errorf("expected 1 redirected, got %d", stats.RedirectedQueries)
	}
}

func TestStatisticsRepo_UnknownAction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormStatisticsRepo(db)

	repo.IncrementCounter("something_weird")

	var stats models.Statistics
	db.First(&stats, 1)

	// Unknown action still increments total
	if stats.TotalQueries != 1 {
		t.Errorf("expected 1 total for unknown action, got %d", stats.TotalQueries)
	}
	if stats.AllowedQueries != 0 || stats.BlockedQueries != 0 || stats.RedirectedQueries != 0 {
		t.Error("unknown action should not increment specific counters")
	}
}

// --- System State Repository ---

func TestSystemStateRepo_GetCreatesDefault(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemStateRepo(db)

	state, err := repo.Get()
	if err != nil {
		t.Fatal(err)
	}
	if !state.DNSEnabled {
		t.Error("expected DNSEnabled=true by default")
	}
	if !state.PolicyEnabled {
		t.Error("expected PolicyEnabled=true by default")
	}
}

func TestSystemStateRepo_SetDNSEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemStateRepo(db)

	// Create initial state
	repo.Get()

	if err := repo.SetDNSEnabled(true); err != nil {
		t.Fatal(err)
	}

	state, _ := repo.Get()
	if !state.DNSEnabled {
		t.Error("expected DNSEnabled=true after setting")
	}
}

func TestSystemStateRepo_SetPolicyEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemStateRepo(db)

	repo.Get()

	if err := repo.SetPolicyEnabled(true); err != nil {
		t.Fatal(err)
	}

	state, _ := repo.Get()
	if !state.PolicyEnabled {
		t.Error("expected PolicyEnabled=true after setting")
	}
}

func TestSystemStateRepo_GetIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSystemStateRepo(db)

	s1, _ := repo.Get()
	s2, _ := repo.Get()
	if s1.ID != s2.ID {
		t.Error("expected same row on repeated Get()")
	}
}

// --- Policy Repository ---

func TestPolicyRepo_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPolicyRepo(db)

	p := &models.Policy{
		ID: "block-ads", Name: "Block Ads", Action: "BLOCK",
		Domains: `["ads.example.com","tracker.com"]`, Priority: 100, Enabled: true,
	}
	if err := repo.Create(p); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID("block-ads")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Block Ads" {
		t.Errorf("expected name 'Block Ads', got %q", got.Name)
	}
	if got.Priority != 100 {
		t.Errorf("expected priority 100, got %d", got.Priority)
	}
}

func TestPolicyRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPolicyRepo(db)

	repo.Create(&models.Policy{ID: "p1", Name: "P1", Action: "BLOCK", Priority: 10, Enabled: true})
	repo.Create(&models.Policy{ID: "p2", Name: "P2", Action: "ALLOW", Priority: 200, Enabled: false})

	list, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(list))
	}
	// Should be ordered by priority desc
	if list[0].ID != "p2" {
		t.Errorf("expected highest priority first, got %s", list[0].ID)
	}
}

func TestPolicyRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPolicyRepo(db)

	repo.Create(&models.Policy{ID: "del-me", Name: "Del", Action: "BLOCK"})

	if err := repo.Delete("del-me"); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GetByID("del-me")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestPolicyRepo_DeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPolicyRepo(db)

	err := repo.Delete("nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent policy")
	}
}

// --- Blocklist Source Methods ---

func TestBlocklistRepo_SourceCRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{
		ID: "steven-black", Name: "StevenBlack", URL: "http://example.com/hosts",
		Format: "hosts", Category: "ads", Enabled: true, CreatedAt: time.Now(),
	}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}

	// List
	sources, err := repo.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	// Get
	got, err := repo.GetSource("steven-black")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "StevenBlack" {
		t.Errorf("expected name StevenBlack, got %q", got.Name)
	}

	// Count entries (should be 0)
	count, err := repo.CountEntriesBySource("steven-black")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 entries, got %d", count)
	}

	// Delete
	if err := repo.DeleteSource("steven-black"); err != nil {
		t.Fatal(err)
	}
	sources, _ = repo.ListSources()
	if len(sources) != 0 {
		t.Error("expected 0 sources after delete")
	}
}

func TestBlocklistRepo_DeleteSourceCascades(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "src1", Name: "Test", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	repo.CreateSource(src)

	entries := []models.BlocklistEntry{
		{Domain: "a.com", SourceID: "src1"},
		{Domain: "b.com", SourceID: "src1"},
	}
	repo.SaveSnapshotWithEntries(*src, "hash", entries)

	// Verify entries exist
	count, _ := repo.CountEntriesBySource("src1")
	if count != 2 {
		t.Fatalf("expected 2 entries before delete, got %d", count)
	}

	// Delete source: should cascade
	repo.DeleteSource("src1")

	count, _ = repo.CountEntriesBySource("src1")
	if count != 0 {
		t.Errorf("expected 0 entries after cascade delete, got %d", count)
	}
}

// GetAllEnabled must exclude entries whose source is disabled, even though
// the rows are still in the DB, or a toggled-off list would keep blocking.
// GetAll (unfiltered) deliberately keeps the old behavior for any other
// future caller.
func TestBlocklistRepo_GetAllEnabled_ExcludesDisabledSources(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	on := &models.BlocklistSource{ID: "on", Name: "On", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	off := &models.BlocklistSource{ID: "off", Name: "Off", URL: "http://y", Format: "hosts", Enabled: false, CreatedAt: time.Now()}
	if err := repo.CreateSource(on); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSource(off); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.BlocklistEntry{Domain: "enabled.com", SourceID: "on"})
	db.Create(&models.BlocklistEntry{Domain: "disabled.com", SourceID: "off"})

	domains, err := repo.GetAllEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "enabled.com" {
		t.Fatalf("GetAllEnabled() = %v, want [enabled.com]", domains)
	}

	// Sanity: GetAll (unfiltered) still returns both. Wiring this method
	// into the DNS hot path instead of GetAllEnabled would reintroduce
	// the toggle-doesn't-block bug.
	all, err := repo.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("GetAll() = %v, want both domains", all)
	}
}

func TestBlocklistRepo_GetAllEnabled_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	domains, err := repo.GetAllEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(domains))
	}
}

// --- Blocklist Signature ---

func TestBlocklistRepo_Signature_StableWhenUnchanged(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}

	sig1, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if sig1 != sig2 {
		t.Errorf("Signature changed with no writes in between: %+v != %+v", sig1, sig2)
	}
}

func TestBlocklistRepo_Signature_ChangesOnCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	before, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("Signature did not change after CreateSource: %+v", after)
	}
	if after.SourceCount != before.SourceCount+1 {
		t.Errorf("SourceCount = %d, want %d", after.SourceCount, before.SourceCount+1)
	}
}

func TestBlocklistRepo_Signature_ChangesOnToggle(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}

	// Mirror what UpdateBlocklist does: mutate the in-memory struct and
	// persist a full row save (UpdateSourceFields), including bumping
	// UpdatedAt as the handler does.
	time.Sleep(time.Millisecond)
	src.Enabled = false
	src.UpdatedAt = time.Now()
	if err := repo.UpdateSourceFields(src); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("Signature did not change after toggling enabled: %+v", after)
	}
	if after.EnabledSourceCount != before.EnabledSourceCount-1 {
		t.Errorf("EnabledSourceCount = %d, want %d", after.EnabledSourceCount, before.EnabledSourceCount-1)
	}
	if after.MaxSourceUpdatedAt == before.MaxSourceUpdatedAt {
		t.Errorf("MaxSourceUpdatedAt did not change: before=%q after=%q", before.MaxSourceUpdatedAt, after.MaxSourceUpdatedAt)
	}
}

func TestBlocklistRepo_Signature_ChangesOnEdit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)
	src.Name = "renamed"
	src.UpdatedAt = time.Now()
	if err := repo.UpdateSourceFields(src); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("Signature did not change after editing name: %+v", after)
	}
}

func TestBlocklistRepo_Signature_ChangesOnDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteSource("s1"); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("Signature did not change after DeleteSource: %+v", after)
	}
	if after.SourceCount != before.SourceCount-1 {
		t.Errorf("SourceCount = %d, want %d", after.SourceCount, before.SourceCount-1)
	}
}

func TestBlocklistRepo_Signature_ChangesOnNewSnapshot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBlocklistRepo(db)

	src := &models.BlocklistSource{ID: "s1", Name: "S1", URL: "http://x", Format: "hosts", Enabled: true, CreatedAt: time.Now()}
	if err := repo.CreateSource(src); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)
	entries := []models.BlocklistEntry{{Domain: "a.com", SourceID: "s1"}}
	if _, err := repo.SaveSnapshotWithEntries(*src, "hash1", entries); err != nil {
		t.Fatal(err)
	}

	after, err := repo.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Errorf("Signature did not change after a new snapshot: %+v", after)
	}
	if after.SnapshotCount != before.SnapshotCount+1 {
		t.Errorf("SnapshotCount = %d, want %d", after.SnapshotCount, before.SnapshotCount+1)
	}
	if after.MaxSnapshotID <= before.MaxSnapshotID {
		t.Errorf("MaxSnapshotID did not advance: before=%d after=%d", before.MaxSnapshotID, after.MaxSnapshotID)
	}
}
