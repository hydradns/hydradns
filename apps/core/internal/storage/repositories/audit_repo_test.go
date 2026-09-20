// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAudit_RecordAndQuery(t *testing.T) {
	db := openAuditTestDB(t)
	repo := NewAuditRepo(db)

	actor := uint(7)
	before := `{"priority":50}`
	after := `{"priority":100}`

	err := repo.Record(&models.AuditEvent{
		ActorID:    &actor,
		Action:     "policy.update",
		Target:     "policy:block-social",
		BeforeJSON: &before,
		AfterJSON:  &after,
		ClientIP:   "192.168.1.38",
		UserAgent:  "hydra-cli/1.0",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}

	got, err := repo.Query(AuditFilter{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	evt := got[0]
	if evt.ActorID == nil || *evt.ActorID != actor {
		t.Errorf("actor: %+v", evt.ActorID)
	}
	if evt.Action != "policy.update" {
		t.Errorf("action: %q", evt.Action)
	}
	if evt.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-filled")
	}
	if evt.BeforeJSON == nil || *evt.BeforeJSON != before {
		t.Error("before json round-trip failed")
	}
	if evt.AfterJSON == nil || *evt.AfterJSON != after {
		t.Error("after json round-trip failed")
	}
}

func TestAudit_Record_NullableActor(t *testing.T) {
	db := openAuditTestDB(t)
	repo := NewAuditRepo(db)

	// System-initiated events (migrations, scheduled jobs) have no actor.
	err := repo.Record(&models.AuditEvent{
		Action:    "system.migrate.admin_singleton",
		Target:    "user:admin@hydradns.local",
		ClientIP:  "127.0.0.1",
		UserAgent: "hydradns/migrate",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	got, _ := repo.Query(AuditFilter{})
	if len(got) != 1 || got[0].ActorID != nil {
		t.Errorf("expected nil actor on system event, got %+v", got[0].ActorID)
	}
}

func TestAudit_Query_Filters(t *testing.T) {
	db := openAuditTestDB(t)
	repo := NewAuditRepo(db)

	a1, a2 := uint(1), uint(2)
	base := time.Now()
	seed := []models.AuditEvent{
		{ActorID: &a1, Action: "policy.create", Target: "policy:a", ClientIP: "1", UserAgent: "ua", CreatedAt: base.Add(-3 * time.Hour)},
		{ActorID: &a2, Action: "policy.delete", Target: "policy:a", ClientIP: "1", UserAgent: "ua", CreatedAt: base.Add(-2 * time.Hour)},
		{ActorID: &a1, Action: "blocklist.create", Target: "blocklist:x", ClientIP: "1", UserAgent: "ua", CreatedAt: base.Add(-1 * time.Hour)},
		{ActorID: &a2, Action: "policy.create", Target: "policy:b", ClientIP: "1", UserAgent: "ua", CreatedAt: base},
	}
	for i := range seed {
		if err := repo.Record(&seed[i]); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	// Filter by actor
	got, _ := repo.Query(AuditFilter{ActorID: &a1})
	if len(got) != 2 {
		t.Errorf("actor filter: got %d, want 2", len(got))
	}

	// Filter by action
	got, _ = repo.Query(AuditFilter{Action: "policy.create"})
	if len(got) != 2 {
		t.Errorf("action filter: got %d, want 2", len(got))
	}

	// Filter by target
	got, _ = repo.Query(AuditFilter{Target: "policy:a"})
	if len(got) != 2 {
		t.Errorf("target filter: got %d, want 2", len(got))
	}

	// Filter by window
	from := base.Add(-90 * time.Minute)
	got, _ = repo.Query(AuditFilter{From: &from})
	if len(got) != 2 {
		t.Errorf("from filter: got %d, want 2 (last 90min)", len(got))
	}

	// Compound filter: actor=2 AND action=policy.create -> only the last row
	got, _ = repo.Query(AuditFilter{ActorID: &a2, Action: "policy.create"})
	if len(got) != 1 || got[0].Target != "policy:b" {
		t.Errorf("compound filter: got %+v", got)
	}

	// Count honours the same filters
	n, _ := repo.Count(AuditFilter{ActorID: &a1})
	if n != 2 {
		t.Errorf("count actor=1: got %d, want 2", n)
	}
}

func TestAudit_Query_OrderingNewestFirst(t *testing.T) {
	db := openAuditTestDB(t)
	repo := NewAuditRepo(db)

	t1 := time.Now().Add(-2 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)
	t3 := time.Now()
	for _, ts := range []time.Time{t1, t2, t3} {
		_ = repo.Record(&models.AuditEvent{
			Action: "x", Target: "y", ClientIP: "1", UserAgent: "ua", CreatedAt: ts,
		})
	}
	got, _ := repo.Query(AuditFilter{})
	if len(got) != 3 {
		t.Fatalf("expected 3 rows")
	}
	if !(got[0].CreatedAt.After(got[1].CreatedAt) && got[1].CreatedAt.After(got[2].CreatedAt)) {
		t.Errorf("expected newest-first ordering, got timestamps %v %v %v", got[0].CreatedAt, got[1].CreatedAt, got[2].CreatedAt)
	}
}
