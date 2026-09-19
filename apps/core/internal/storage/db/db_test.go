// SPDX-License-Identifier: GPL-3.0-or-later
package db

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	return db
}

// TestMigrate_FreshInstall ensures AutoMigrate on an empty database creates
// all tables and does not spuriously create users.
func TestMigrate_FreshInstall(t *testing.T) {
	db := openTestDB(t)

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("fresh install: expected 0 users, got %d", userCount)
	}
}

// TestMigrate_LegacySingleton simulates a pre-RBAC database that already
// has an AdminCredential, and verifies the migration copies it into a
// User + Token with the API key preserved as a hash.
func TestMigrate_LegacySingleton(t *testing.T) {
	db := openTestDB(t)

	// First migration pass creates the schema so we can pre-populate
	// the legacy singleton row the way a pre-RBAC binary would have.
	if err := db.AutoMigrate(&models.AdminCredential{}); err != nil {
		t.Fatalf("pre-migrate legacy table: %v", err)
	}
	legacyKey := "d7058cc8-bb91-41b6-ae64-69f1c0f93f48"
	admin := models.AdminCredential{
		PasswordHash: "$2a$10$fakebcrypthashforthepurposesofthistest",
		APIKey:       legacyKey,
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("seed legacy admin: %v", err)
	}

	// Run the real migration.
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Exactly one user should now exist, with admin role.
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 migrated user, got %d", len(users))
	}
	u := users[0]
	if u.Role != models.RoleAdmin {
		t.Errorf("role: got %q, want %q", u.Role, models.RoleAdmin)
	}
	if u.PasswordHash != admin.PasswordHash {
		t.Errorf("password hash mismatch after migration")
	}
	if u.Email == "" {
		t.Errorf("migrated user has empty email")
	}

	// The legacy API key must still authenticate: look up by the hash
	// of the plaintext and verify it resolves to the migrated user.
	sum := sha256.Sum256([]byte(legacyKey))
	wantHash := hex.EncodeToString(sum[:])
	var tok models.Token
	if err := db.Where("hash = ?", wantHash).First(&tok).Error; err != nil {
		t.Fatalf("legacy key did not migrate to a Token row: %v", err)
	}
	if tok.UserID != u.ID {
		t.Errorf("token owner: got %d, want %d", tok.UserID, u.ID)
	}
	if tok.ExpiresAt != nil {
		t.Errorf("migrated token should have no expiry, got %v", tok.ExpiresAt)
	}
	if tok.RevokedAt != nil {
		t.Errorf("migrated token should not be revoked")
	}
}

// TestMigrate_Idempotent runs the migration twice against a database that
// already has a migrated user. The second pass must not create duplicates
// or fail.
func TestMigrate_Idempotent(t *testing.T) {
	db := openTestDB(t)

	if err := db.AutoMigrate(&models.AdminCredential{}); err != nil {
		t.Fatalf("pre-migrate legacy table: %v", err)
	}
	if err := db.Create(&models.AdminCredential{PasswordHash: "x", APIKey: "key-xyz"}).Error; err != nil {
		t.Fatalf("seed legacy admin: %v", err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("second migrate (should be no-op): %v", err)
	}

	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount != 1 {
		t.Errorf("expected 1 user after two migrations, got %d", userCount)
	}

	var tokenCount int64
	db.Model(&models.Token{}).Count(&tokenCount)
	if tokenCount != 1 {
		t.Errorf("expected 1 token after two migrations, got %d", tokenCount)
	}
}

// TestToken_Valid covers the three failure modes of Token.Valid.
func TestToken_Valid(t *testing.T) {
	now := timeMustParse(t, "2026-04-23T12:00:00Z")

	cases := []struct {
		name string
		tok  models.Token
		want bool
	}{
		{
			name: "no expiry, no revocation",
			tok:  models.Token{},
			want: true,
		},
		{
			name: "future expiry",
			tok:  models.Token{ExpiresAt: timePtr(timeMustParse(t, "2026-05-01T00:00:00Z"))},
			want: true,
		},
		{
			name: "past expiry",
			tok:  models.Token{ExpiresAt: timePtr(timeMustParse(t, "2026-04-01T00:00:00Z"))},
			want: false,
		},
		{
			name: "revoked",
			tok:  models.Token{RevokedAt: timePtr(timeMustParse(t, "2026-04-10T00:00:00Z"))},
			want: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := tc.tok.Valid(now)
			if got != tc.want {
				t.Errorf("Valid(): got %v, want %v", got, tc.want)
			}
		})
	}
}
