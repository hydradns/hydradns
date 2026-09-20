// SPDX-License-Identifier: GPL-3.0-or-later
package models

import "time"

type BlocklistSource struct {
	ID        string `gorm:"primaryKey;size:64"`
	Name      string
	URL       string
	Format    string
	Category  string
	Enabled   bool
	Priority  int
	UpdatedAt time.Time
	ETag      string
	LastHash  string
	CreatedAt time.Time
}

type BlocklistSnapshot struct {
	ID        uint   `gorm:"primaryKey"`
	SourceID  string `gorm:"index"`
	CreatedAt time.Time
	Size      int
	Checksum  string
	Path      string // optional file path if persisted to disk
}

type BlocklistEntry struct {
	ID         uint   `gorm:"primaryKey"`
	SnapshotID uint   `gorm:"index"`
	Domain     string `gorm:"index;size:255"`
	// SourceID is indexed: both DeleteSource and SaveSnapshotWithEntries's
	// replace-on-ingest DELETE filter by this column, and without an index
	// each is a full table scan over a table that can hold millions of
	// rows on a full Pi blocklist install. On an existing large
	// blocklist_entries table, AutoMigrate runs a CREATE INDEX for this
	// the first time a build with this tag starts, which holds a write
	// lock for the duration of the index build. Both controlplane and
	// dataplane call AutoMigrate against the same single-writer SQLite
	// file at startup, so this shares the same concurrent-AutoMigrate
	// risk as db.go's busy_timeout (see storage/db/db.go).
	SourceID  string `gorm:"index"`
	Category  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
