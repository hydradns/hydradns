// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"database/sql"
	"strings"
	"time"

	"github.com/hydradns/hydra-core/internal/storage/models"
	"gorm.io/gorm"
)

// Interface (clean, mockable)
type BlocklistRepository interface {
	SaveSnapshotWithEntries(src models.BlocklistSource, checksum string, entries []models.BlocklistEntry) (models.BlocklistSnapshot, error)
	GetAll() ([]string, error)
	// GetAllEnabled returns domains from enabled sources only. This is what
	// the DNS hot path's in-memory set should be built from: a disabled
	// source's rows remain in the DB (only DeleteSource removes them), so
	// GetAll (unfiltered) would keep blocking through a source the
	// dashboard/CLI/MCP already toggled off. See blocklist.Engine.List.
	GetAllEnabled() ([]string, error)
	IsBlocked(domain string) (bool, error)
	ListSources() ([]models.BlocklistSource, error)
	GetSource(id string) (*models.BlocklistSource, error)
	CreateSource(src *models.BlocklistSource) error
	// UpdateSourceFields persists an in-memory-edited BlocklistSource as a
	// full row replace (like PolicyRepo.Update). Named distinctly from
	// blocklist.Engine.UpdateSource (which fetches+parses+persists a
	// snapshot) to avoid confusion where both are used side by side in
	// the blocklists handler.
	UpdateSourceFields(src *models.BlocklistSource) error
	DeleteSource(id string) error
	CountEntriesBySource(sourceID string) (int64, error)
	CountEntriesGroupedBySource() (map[string]int64, error)
	// Signature returns a cheap fingerprint of blocklist DB state, used by
	// the dataplane's poll loop to detect whether anything relevant to the
	// in-memory blocklist set changed (source added/edited/toggled/deleted,
	// or a new snapshot ingested) without paying the cost of reading the
	// full entries table on every poll tick.
	Signature() (BlocklistSignature, error)
}

// BlocklistSignature is a cheap, comparable (==) fingerprint of blocklist
// state. Every field is chosen to change on at least one relevant write
// path so the dataplane's poll loop never misses a change:
//
//   - SourceCount        — create (+1) / delete (-1) a source
//   - EnabledSourceCount — toggle enabled on/off (belt-and-suspenders: a
//     toggle also bumps MaxSourceUpdatedAt, see UpdateBlocklist in the
//     control-plane handler, but this field makes the "enabled" dimension
//     explicit and catches it even if that ever changes)
//   - MaxSourceUpdatedAt — edit (name/url/format/category), toggle, and
//     ingestion completing (SaveSnapshotWithEntries also stamps the
//     source's UpdatedAt in the same transaction as the new snapshot)
//   - SnapshotCount / MaxSnapshotID — a new snapshot ingested (create's
//     background fetch, an edit's re-fetch, or the periodic refresh);
//     MaxSnapshotID also catches the edge case where a delete removes the
//     single most-recently-updated source (SourceCount already catches
//     the delete itself, this is redundant-but-cheap defense in depth)
type BlocklistSignature struct {
	SourceCount        int64
	EnabledSourceCount int64
	// MaxSourceUpdatedAt is the raw text the DB driver returns for
	// MAX(updated_at) (the pure-Go sqlite driver round-trips time.Time as
	// text, not a value sql.Scan can convert into time.Time from a plain
	// aggregate query). Only used for equality comparison, never parsed —
	// an opaque token is all a change-detection signature needs.
	MaxSourceUpdatedAt string
	SnapshotCount      int64
	MaxSnapshotID      uint
}

// Implementation
type BlocklistRepo struct {
	db *gorm.DB
}

func NewBlocklistRepo(db *gorm.DB) *BlocklistRepo {
	return &BlocklistRepo{db: db}
}

func (r *BlocklistRepo) IsBlocked(domain string) (bool, error) {
	// Normalize domain (lowercase, remove trailing dot)
	d := strings.TrimSuffix(strings.ToLower(domain), ".")

	// Check exact match + parent domains (www.ads.google.com → ads.google.com → google.com)
	parts := strings.Split(d, ".")
	candidates := make([]string, 0, len(parts)-1)
	for i := 0; i < len(parts)-1; i++ {
		candidates = append(candidates, strings.Join(parts[i:], "."))
	}

	var count int64
	err := r.db.Model(&models.BlocklistEntry{}).
		Where("domain IN ?", candidates).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *BlocklistRepo) GetAll() ([]string, error) {
	var domains []string
	if err := r.db.Model(&models.BlocklistEntry{}).Pluck("domain", &domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

// GetAllEnabled returns domains belonging only to sources currently marked
// enabled. See the interface doc: unlike GetAll, this excludes a disabled
// source's entries even though its rows are still in the DB.
func (r *BlocklistRepo) GetAllEnabled() ([]string, error) {
	var domains []string
	err := r.db.Model(&models.BlocklistEntry{}).
		Joins("JOIN blocklist_sources ON blocklist_sources.id = blocklist_entries.source_id").
		Where("blocklist_sources.enabled = ?", true).
		Pluck("blocklist_entries.domain", &domains).Error
	if err != nil {
		return nil, err
	}
	return domains, nil
}

func (r *BlocklistRepo) SaveSnapshotWithEntries(src models.BlocklistSource, checksum string, entries []models.BlocklistEntry) (models.BlocklistSnapshot, error) {
	tx := r.db.Begin()
	snapshot := models.BlocklistSnapshot{
		SourceID: src.ID, CreatedAt: time.Now(), Size: len(entries), Checksum: checksum,
	}
	if err := tx.Create(&snapshot).Error; err != nil {
		tx.Rollback()
		return snapshot, err
	}
	for i := range entries {
		entries[i].SnapshotID = snapshot.ID
		if err := tx.Create(&entries[i]).Error; err != nil {
			tx.Rollback()
			return snapshot, err
		}
	}
	// update source metadata
	src.UpdatedAt = time.Now()
	if err := tx.Save(&src).Error; err != nil {
		tx.Rollback()
		return snapshot, err
	}
	if err := tx.Commit().Error; err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (r *BlocklistRepo) ListSources() ([]models.BlocklistSource, error) {
	var sources []models.BlocklistSource
	err := r.db.Order("created_at desc").Find(&sources).Error
	return sources, err
}

func (r *BlocklistRepo) GetSource(id string) (*models.BlocklistSource, error) {
	var src models.BlocklistSource
	err := r.db.First(&src, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &src, nil
}

func (r *BlocklistRepo) CreateSource(src *models.BlocklistSource) error {
	return r.db.Create(src).Error
}

// UpdateSourceFields does a full row save (not a partial GORM Updates()
// map) so an explicit false/"" value — e.g. disabling a source — is
// actually persisted. GORM's struct-based Updates() silently skips
// zero-valued fields, which would make it impossible to ever turn
// Enabled back to false; callers are expected to have merged their
// partial request onto the existing row first (see handlers.UpdateBlocklist).
func (r *BlocklistRepo) UpdateSourceFields(src *models.BlocklistSource) error {
	return r.db.Save(src).Error
}

func (r *BlocklistRepo) DeleteSource(id string) error {
	tx := r.db.Begin()
	defer tx.Rollback() // no-op after commit

	if err := tx.Where("source_id = ?", id).Delete(&models.BlocklistEntry{}).Error; err != nil {
		return err
	}
	if err := tx.Where("source_id = ?", id).Delete(&models.BlocklistSnapshot{}).Error; err != nil {
		return err
	}
	result := tx.Delete(&models.BlocklistSource{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return tx.Commit().Error
}

func (r *BlocklistRepo) CountEntriesBySource(sourceID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.BlocklistEntry{}).Where("source_id = ?", sourceID).Count(&count).Error
	return count, err
}

// CountEntriesGroupedBySource returns domain counts keyed by source ID in a single query.
func (r *BlocklistRepo) CountEntriesGroupedBySource() (map[string]int64, error) {
	type result struct {
		SourceID string
		Count    int64
	}
	var results []result
	err := r.db.Model(&models.BlocklistEntry{}).
		Select("source_id, count(*) as count").
		Group("source_id").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(results))
	for _, r := range results {
		counts[r.SourceID] = r.Count
	}
	return counts, nil
}

// Signature computes BlocklistSignature with two cheap aggregate queries
// over blocklist_sources and blocklist_snapshots — both tables are small
// (one row per configured source / per fetch), so this stays fast even
// when blocklist_entries holds millions of rows for a Raspberry Pi's worth
// of blocklists. It never touches blocklist_entries.
func (r *BlocklistRepo) Signature() (BlocklistSignature, error) {
	var sig BlocklistSignature

	type sourceAgg struct {
		Count        int64
		EnabledCount int64
		MaxUpdatedAt sql.NullString
	}
	var sAgg sourceAgg
	if err := r.db.Model(&models.BlocklistSource{}).
		Select("COUNT(*) AS count, COALESCE(SUM(CASE WHEN enabled THEN 1 ELSE 0 END), 0) AS enabled_count, MAX(updated_at) AS max_updated_at").
		Scan(&sAgg).Error; err != nil {
		return sig, err
	}
	sig.SourceCount = sAgg.Count
	sig.EnabledSourceCount = sAgg.EnabledCount
	sig.MaxSourceUpdatedAt = sAgg.MaxUpdatedAt.String

	type snapshotAgg struct {
		Count int64
		MaxID uint
	}
	var snAgg snapshotAgg
	if err := r.db.Model(&models.BlocklistSnapshot{}).
		Select("COUNT(*) AS count, COALESCE(MAX(id), 0) AS max_id").
		Scan(&snAgg).Error; err != nil {
		return sig, err
	}
	sig.SnapshotCount = snAgg.Count
	sig.MaxSnapshotID = snAgg.MaxID

	return sig, nil
}
