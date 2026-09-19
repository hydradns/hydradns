// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUser_Create_HappyPath(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	u, err := repo.Create("Alice@Example.COM", "hash", models.RoleOperator)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("email should be lowercased and trimmed, got %q", u.Email)
	}
	if u.ID == 0 {
		t.Error("ID should be assigned")
	}
}

func TestUser_Create_RejectsInvalidRole(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	_, err := repo.Create("x@x.com", "h", "superuser")
	if err == nil || !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("expected invalid role error, got %v", err)
	}
}

func TestUser_Create_RejectsEmptyFields(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	if _, err := repo.Create("", "h", models.RoleAdmin); err == nil {
		t.Error("empty email should fail")
	}
	if _, err := repo.Create("x@x.com", "", models.RoleAdmin); err == nil {
		t.Error("empty password hash should fail")
	}
}

func TestUser_DuplicateEmail(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	if _, err := repo.Create("dup@x.com", "h", models.RoleAdmin); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := repo.Create("DUP@x.com", "h2", models.RoleAdmin); err == nil {
		t.Error("duplicate email (case-insensitive) should violate UNIQUE")
	}
}

func TestUser_UpdateRole(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	u, _ := repo.Create("u@x.com", "h", models.RoleOperator)
	if err := repo.UpdateRole(u.ID, models.RoleAdmin); err != nil {
		t.Fatalf("promote: %v", err)
	}
	got, _ := repo.Get(u.ID)
	if got.Role != models.RoleAdmin {
		t.Errorf("role: got %q, want admin", got.Role)
	}
	if err := repo.UpdateRole(u.ID, "nonsense"); err == nil {
		t.Error("invalid role on update should fail")
	}
}

func TestUser_SetDisabled(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	u, _ := repo.Create("u@x.com", "h", models.RoleAdmin)
	if err := repo.SetDisabled(u.ID, true); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ := repo.Get(u.ID)
	if !got.Disabled {
		t.Error("user should be disabled")
	}
}

func TestUser_GetByEmail_CaseInsensitive(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	if _, err := repo.Create("Mixed@Case.org", "h", models.RoleAdmin); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.GetByEmail("mixed@case.org")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got.Email != "mixed@case.org" {
		t.Errorf("email: got %q", got.Email)
	}
}

func TestUser_Count(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewUserRepo(db)

	n, _ := repo.Count()
	if n != 0 {
		t.Errorf("empty db: got %d", n)
	}
	repo.Create("a@x.com", "h", models.RoleAdmin)
	repo.Create("b@x.com", "h", models.RoleOperator)
	n, _ = repo.Count()
	if n != 2 {
		t.Errorf("after two creates: got %d", n)
	}
}
