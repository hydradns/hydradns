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

func openTokenTestDB(t *testing.T) (*gorm.DB, *models.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Token{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	u := &models.User{Email: "alice@example.com", PasswordHash: "x", Role: models.RoleOperator}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return db, u
}

func TestToken_CreateForUser_ReturnsUsablePlaintext(t *testing.T) {
	db, u := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	plaintext, tok, err := repo.CreateForUser(u.ID, "laptop", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if plaintext == "" {
		t.Fatal("plaintext should be returned")
	}
	if tok.Hash == plaintext {
		t.Fatal("stored hash must not equal plaintext")
	}
	if tok.ExpiresAt == nil {
		t.Fatal("default expiry should apply when expiry <= 0")
	}

	gotTok, gotUser, err := repo.ResolveActive(plaintext)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gotTok == nil || gotUser == nil {
		t.Fatal("resolve should succeed for a just-minted token")
	}
	if gotUser.ID != u.ID {
		t.Errorf("resolved user: got %d, want %d", gotUser.ID, u.ID)
	}
	if gotTok.LastUsedAt == nil {
		t.Error("LastUsedAt should be bumped after ResolveActive")
	}
}

func TestToken_Create_RequiresLabel(t *testing.T) {
	db, u := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	_, _, err := repo.CreateForUser(u.ID, "", 0)
	if err == nil {
		t.Fatal("expected error for empty label, got nil")
	}
}

func TestToken_Resolve_RejectsRevoked(t *testing.T) {
	db, u := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	plaintext, tok, _ := repo.CreateForUser(u.ID, "laptop", time.Hour)
	if err := repo.Revoke(tok.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	gotTok, gotUser, err := repo.ResolveActive(plaintext)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if gotTok != nil || gotUser != nil {
		t.Fatal("revoked token must not resolve")
	}
}

func TestToken_Resolve_RejectsExpired(t *testing.T) {
	db, u := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	// Create via CreateUnexpiring then back-date ExpiresAt so we don't
	// have to wait real wall-clock time.
	plaintext, tok, err := repo.CreateUnexpiring(u.ID, "laptop")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := db.Model(&models.Token{}).Where("id = ?", tok.ID).Update("expires_at", past).Error; err != nil {
		t.Fatalf("backdate expiry: %v", err)
	}

	gotTok, _, _ := repo.ResolveActive(plaintext)
	if gotTok != nil {
		t.Fatal("expired token must not resolve")
	}
}

func TestToken_Resolve_RejectsDisabledUser(t *testing.T) {
	db, u := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	plaintext, _, _ := repo.CreateForUser(u.ID, "laptop", time.Hour)
	if err := db.Model(&models.User{}).Where("id = ?", u.ID).Update("disabled", true).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	gotTok, _, _ := repo.ResolveActive(plaintext)
	if gotTok != nil {
		t.Fatal("token belonging to disabled user must not resolve")
	}
}

func TestToken_Resolve_MissReturnsNilNil(t *testing.T) {
	db, _ := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	tok, user, err := repo.ResolveActive("nonsense-token-never-created")
	if err != nil {
		t.Errorf("miss should not return an error, got %v", err)
	}
	if tok != nil || user != nil {
		t.Error("miss should return nil token and nil user")
	}
}

func TestToken_Resolve_EmptyPlaintext(t *testing.T) {
	db, _ := openTokenTestDB(t)
	repo := NewTokenRepo(db)

	tok, user, err := repo.ResolveActive("")
	if err != nil || tok != nil || user != nil {
		t.Errorf("empty plaintext should resolve to nil/nil/nil, got %v %v %v", tok, user, err)
	}
}

func TestToken_HashToken_DeterministicAndMatchesModels(t *testing.T) {
	// Guard the single source of truth for the hash algorithm.
	repoHash := HashToken("abc123")
	modelHash := models.HashToken("abc123")
	if repoHash != modelHash {
		t.Fatalf("repositories.HashToken and models.HashToken diverged: %q vs %q", repoHash, modelHash)
	}
}
