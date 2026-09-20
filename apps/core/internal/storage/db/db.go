package db

import (
	"log"
	"time"

	"github.com/glebarez/sqlite" // <-- use this, pure-Go driver
	"github.com/hydradns/hydra-core/internal/storage/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init initializes SQLite with sane defaults.
func InitDB(path string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // reduce noise
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Connection pool tuning
	// SPDX-License-Identifier: GPL-3.0-or-later
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // SQLite is single-writer
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Enable WAL for better concurrency
	db.Exec("PRAGMA journal_mode=WAL;")

	// busy_timeout makes SQLite retry internally (up to this many ms)
	// instead of immediately returning SQLITE_BUSY when another process
	// holds the write lock. Without it, the default is 0, no retry at
	// all. This matters specifically at migration time: cmd/controlplane
	// and cmd/dataplane both call InitDB (and therefore AutoMigrate)
	// against the same single-writer SQLite file, and on the combined
	// `core` container they can start simultaneously. AutoMigrate adding
	// an index to dns_queries (see models.DNSQuery's Action field) is a
	// CREATE INDEX over a table that can hold up to ~1M rows on an
	// existing install, a real write that can take more than an instant.
	// 30s comfortably covers that on a Pi's SD card; the alternative
	// (only one process ever migrates) would require the dataplane to
	// wait/retry opening until the schema is ready, which is a change to
	// cmd/dataplane/main.go outside this package's scope.
	db.Exec("PRAGMA busy_timeout=30000;")

	// Run migrations
	if err := migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Seed singleton rows that the runtime expects to exist. Idempotent.
	if err := db.Exec(
		"INSERT OR IGNORE INTO statistics (id, total_queries, blocked_queries, allowed_queries, redirected_queries, updated_at) VALUES (1, 0, 0, 0, 0, ?)",
		time.Now(),
	).Error; err != nil {
		log.Fatalf("statistics seed failed: %v", err)
	}

	log.Println("Database connection established")

	DB = db
	return DB
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Policy{},
		&models.DNSQuery{},
		&models.DomainPolicy{},
		&models.Action{},
		&models.Category{},
		&models.Statistics{},
		&models.SystemState{},
		&models.BlocklistSource{},
		&models.BlocklistSnapshot{},
		&models.BlocklistEntry{},
		&models.AdminCredential{},
		&models.User{},
		&models.Token{},
		&models.AuditEvent{},
	); err != nil {
		return err
	}
	return migrateAdminSingletonToUser(db)
}

// migrateAdminSingletonToUser copies an existing AdminCredential singleton
// into the new User + Token tables on first boot of a binary that knows
// about RBAC. The migration is idempotent: once the users table is
// non-empty, this function does nothing.
//
// The existing UUID API key is preserved as a hashed Token with no expiry,
// so existing CLI and dashboard sessions keep working through the upgrade.
// Operators can rename the placeholder email on first login.
func migrateAdminSingletonToUser(db *gorm.DB) error {
	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return nil // already migrated, nothing to do
	}

	var admin models.AdminCredential
	if err := db.First(&admin).Error; err != nil {
		// No singleton to migrate. Fresh install - setup wizard will
		// create the first user. Not an error.
		return nil
	}

	user := models.User{
		Email:        "admin@hydradns.local",
		PasswordHash: admin.PasswordHash,
		Role:         models.RoleAdmin,
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	token := models.Token{
		UserID: user.ID,
		Hash:   models.HashToken(admin.APIKey),
		Label:  "migrated-from-singleton",
		// No ExpiresAt: legacy tokens keep working indefinitely until
		// the admin chooses to rotate.
	}
	if err := db.Create(&token).Error; err != nil {
		return err
	}

	now := time.Now()
	sys := models.AuditEvent{
		Action:    "system.migrate.admin_singleton",
		Target:    "user:" + user.Email,
		ClientIP:  "127.0.0.1",
		UserAgent: "hydradns/migrate",
		CreatedAt: now,
	}
	_ = db.Create(&sys).Error // audit failure is observational, do not fail the migration
	log.Printf("migrated legacy admin singleton to user %q", user.Email)
	return nil
}
