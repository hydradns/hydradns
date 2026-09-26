// SPDX-License-Identifier: Apache-2.0
package demoseed

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.Token{},
		&models.Policy{}, &models.DNSQuery{}, &models.Statistics{},
		&models.BlocklistSource{}, &models.BlocklistSnapshot{}, &models.BlocklistEntry{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The statistics singleton is normally seeded by db.InitDB; replicate
	// that here since these tests build their own in-memory DB directly.
	if err := db.Exec(
		"INSERT INTO statistics (id, total_queries, blocked_queries, allowed_queries, redirected_queries) VALUES (1, 0, 0, 0, 0)",
	).Error; err != nil {
		t.Fatalf("seed statistics singleton: %v", err)
	}
	return db
}

func newTestStore(db *gorm.DB) *repositories.Store {
	return repositories.NewStore(db)
}

func TestEnsureDemoUser_CreatesOnEmptyDB(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := EnsureDemoUser(store); err != nil {
		t.Fatalf("EnsureDemoUser: %v", err)
	}

	u, err := store.Users.GetByEmail(DemoUserEmail)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if u.Role != models.RoleReadOnly {
		t.Errorf("demo user role = %q, want %q", u.Role, models.RoleReadOnly)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(DemoUserPassword)); err != nil {
		t.Errorf("demo user password hash does not match DemoUserPassword: %v", err)
	}
}

func TestEnsureDemoUser_IdempotentOnRestart(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := EnsureDemoUser(store); err != nil {
		t.Fatalf("first EnsureDemoUser: %v", err)
	}
	if err := EnsureDemoUser(store); err != nil {
		t.Fatalf("second EnsureDemoUser (restart) should be a no-op, got: %v", err)
	}

	n, err := store.Users.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("user count after two EnsureDemoUser calls = %d, want 1 (no duplicate)", n)
	}
}

func TestEnsureDemoUser_RefusesNonDemoData(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if _, err := store.Users.Create("real-admin@example.com", "hash", models.RoleAdmin); err != nil {
		t.Fatalf("seed real admin: %v", err)
	}

	if err := EnsureDemoUser(store); err == nil {
		t.Error("expected EnsureDemoUser to refuse a database with a pre-existing non-demo user, got nil error")
	}
}

func TestEnsureDemoUser_RefusesMultipleUsers(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := EnsureDemoUser(store); err != nil {
		t.Fatalf("first EnsureDemoUser: %v", err)
	}
	if _, err := store.Users.Create("someone-else@example.com", "hash", models.RoleOperator); err != nil {
		t.Fatalf("seed second user: %v", err)
	}

	if err := EnsureDemoUser(store); err == nil {
		t.Error("expected EnsureDemoUser to refuse a database with more than the demo user, got nil error")
	}
}

// TestEnsureDemoUser_RefusesExistingQueryLogHistoryWithNoUsers covers a DB
// with query-log rows but zero users (a pre-RBAC volume, or one where
// migration hasn't run): that is not a fresh demo volume, even though
// "zero users" alone would let it through, and proceeding would hand that
// history to the periodic Refresh loop, which deletes dns_queries on its
// very first tick.
func TestEnsureDemoUser_RefusesExistingQueryLogHistoryWithNoUsers(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := store.QueryLogs.Save(&models.DNSQuery{Domain: "real-device.example", ClientIP: "10.0.0.5", Action: "allow"}); err != nil {
		t.Fatalf("seed real query log row: %v", err)
	}

	if err := EnsureDemoUser(store); err == nil {
		t.Error("expected EnsureDemoUser to refuse a database with query log history but no users")
	}

	n, err := store.Users.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 0 {
		t.Errorf("expected no demo user to be created when refusing, got %d users", n)
	}
}

func TestSeedIfEmpty_PopulatesEverything(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := SeedIfEmpty(store); err != nil {
		t.Fatalf("SeedIfEmpty: %v", err)
	}

	n, err := store.QueryLogs.Count()
	if err != nil {
		t.Fatalf("QueryLogs.Count: %v", err)
	}
	if n != totalDemoQueryRows {
		t.Errorf("query log rows = %d, want %d", n, totalDemoQueryRows)
	}

	policies, err := store.Policies.List()
	if err != nil {
		t.Fatalf("Policies.List: %v", err)
	}
	if len(policies) == 0 {
		t.Error("expected at least one seeded policy")
	}

	sources, err := store.Blocklist.ListSources()
	if err != nil {
		t.Fatalf("Blocklist.ListSources: %v", err)
	}
	if len(sources) == 0 {
		t.Error("expected at least one seeded blocklist source")
	}
	for _, src := range sources {
		count, err := store.Blocklist.CountEntriesBySource(src.ID)
		if err != nil {
			t.Fatalf("CountEntriesBySource(%s): %v", src.ID, err)
		}
		if count == 0 {
			t.Errorf("blocklist source %s has zero entries, want a plausible non-zero count", src.ID)
		}
	}

	stats, err := store.Statistics.ListRecent(1)
	if err != nil || len(stats) == 0 {
		t.Fatalf("Statistics.ListRecent: %v", err)
	}
	if stats[0].TotalQueries != uint64(totalDemoQueryRows) {
		t.Errorf("statistics.total_queries = %d, want %d", stats[0].TotalQueries, totalDemoQueryRows)
	}
}

func TestSeedIfEmpty_DoesNotDuplicateOnRestart(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := SeedIfEmpty(store); err != nil {
		t.Fatalf("first SeedIfEmpty: %v", err)
	}
	if err := SeedIfEmpty(store); err != nil {
		t.Fatalf("second SeedIfEmpty (restart): %v", err)
	}

	n, err := store.QueryLogs.Count()
	if err != nil {
		t.Fatalf("QueryLogs.Count: %v", err)
	}
	if n != totalDemoQueryRows {
		t.Errorf("query log rows after two seed calls = %d, want %d (no duplication)", n, totalDemoQueryRows)
	}
}

func TestRefresh_ReanchorsWithoutDuplicatingRowsOrPolicies(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := SeedIfEmpty(store); err != nil {
		t.Fatalf("SeedIfEmpty: %v", err)
	}
	policiesBefore, _ := store.Policies.List()
	sourcesBefore, _ := store.Blocklist.ListSources()

	if err := Refresh(db); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	n, err := store.QueryLogs.Count()
	if err != nil {
		t.Fatalf("QueryLogs.Count: %v", err)
	}
	if n != totalDemoQueryRows {
		t.Errorf("query log rows after Refresh = %d, want %d (regenerated, not appended)", n, totalDemoQueryRows)
	}

	policiesAfter, _ := store.Policies.List()
	sourcesAfter, _ := store.Blocklist.ListSources()
	if len(policiesAfter) != len(policiesBefore) {
		t.Errorf("policy count changed across Refresh: %d -> %d (should be untouched)", len(policiesBefore), len(policiesAfter))
	}
	if len(sourcesAfter) != len(sourcesBefore) {
		t.Errorf("blocklist source count changed across Refresh: %d -> %d (should be untouched)", len(sourcesBefore), len(sourcesAfter))
	}

	stats, err := store.Statistics.ListRecent(1)
	if err != nil || len(stats) == 0 {
		t.Fatalf("Statistics.ListRecent: %v", err)
	}
	if stats[0].TotalQueries != uint64(totalDemoQueryRows) {
		t.Errorf("statistics.total_queries after Refresh = %d, want %d (reset then rebuilt, not accumulated)", stats[0].TotalQueries, totalDemoQueryRows)
	}
}

// TestRefresh_FailurePartwayThroughRollsBackTheDelete verifies the
// DNSQuery delete and the statistics reset are one atomic unit: without
// that, a failure between them could leave the query log deleted but
// statistics untouched (or vice versa), and a reader in that window would
// see an empty table. Forcing the statistics step to fail (by dropping the
// table it targets) and then asserting the delete never took effect proves
// both writes commit or roll back together.
func TestRefresh_FailurePartwayThroughRollsBackTheDelete(t *testing.T) {
	db := openSeedTestDB(t)
	store := newTestStore(db)

	if err := SeedIfEmpty(store); err != nil {
		t.Fatalf("SeedIfEmpty: %v", err)
	}
	before, err := store.QueryLogs.Count()
	if err != nil || before == 0 {
		t.Fatalf("expected seeded rows before Refresh, count=%d err=%v", before, err)
	}

	// Make the statistics-reset step (the second write inside the
	// transaction) fail, so Refresh must roll back the delete that already
	// ran as the first write.
	if err := db.Exec("DROP TABLE statistics").Error; err != nil {
		t.Fatalf("test setup: drop statistics table: %v", err)
	}

	if err := Refresh(db); err == nil {
		t.Fatal("expected Refresh to fail once the statistics table is gone")
	}

	after, err := store.QueryLogs.Count()
	if err != nil {
		t.Fatalf("QueryLogs.Count after failed Refresh: %v", err)
	}
	if after != before {
		t.Errorf("expected the delete to roll back on failure (count unchanged at %d), got %d", before, after)
	}
}
