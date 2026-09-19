// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"time"

	"github.com/lopster568/phantomDNS/internal/logger"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"gorm.io/gorm"
)

// Interface (clean, mockable)
type QueryLogRepository interface {
	Save(query *models.DNSQuery) error
	SaveBatch(queries []*models.DNSQuery) error
	ListRecent(limit int) ([]models.DNSQuery, error)
	DeleteOlderThan(cutoff time.Time) (int64, error)
	EnforceRowCap(maxRows int64) (int64, error)
	Count() (int64, error)
}

// Implementation
type GormQueryLogRepo struct {
	db *gorm.DB
}

func NewGormQueryLogRepo(db *gorm.DB) *GormQueryLogRepo {
	return &GormQueryLogRepo{db: db}
}

func (r *GormQueryLogRepo) Save(query *models.DNSQuery) error {
	query.Timestamp = time.Now()
	logger.Log.Debug("Saving DNS query log")
	logger.Log.Debug("query", query)
	return r.db.Create(query).Error
}

// SaveBatch inserts many query logs in a single multi-row transaction.
// Used by the async log writer so a high query rate collapses into few
// DB round-trips instead of one INSERT (and one fsync) per query.
func (r *GormQueryLogRepo) SaveBatch(queries []*models.DNSQuery) error {
	if len(queries) == 0 {
		return nil
	}
	now := time.Now()
	for _, q := range queries {
		if q.Timestamp.IsZero() {
			q.Timestamp = now
		}
	}
	return r.db.CreateInBatches(queries, 200).Error
}

func (r *GormQueryLogRepo) ListRecent(limit int) ([]models.DNSQuery, error) {
	var queries []models.DNSQuery
	err := r.db.Order("timestamp desc").Limit(limit).Find(&queries).Error
	return queries, err
}

// Count returns the number of rows in the query log table.
func (r *GormQueryLogRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.DNSQuery{}).Count(&n).Error
	return n, err
}

// DeleteOlderThan removes query logs with a timestamp before cutoff and
// returns the number of rows deleted. Timestamp is indexed, so this is a
// ranged delete, not a full scan.
func (r *GormQueryLogRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	res := r.db.Where("timestamp < ?", cutoff).Delete(&models.DNSQuery{})
	return res.RowsAffected, res.Error
}

// EnforceRowCap keeps at most maxRows of the newest query logs, deleting
// the oldest beyond that. This is disk insurance: a sustained query rate
// can exceed any time-window estimate, and the Pi's SD card must not fill.
// IDs are monotonic, so "newest" == "highest id". Returns rows deleted.
func (r *GormQueryLogRepo) EnforceRowCap(maxRows int64) (int64, error) {
	if maxRows <= 0 {
		return 0, nil
	}
	// Find the id of the maxRows-th newest row; everything with a smaller
	// id is surplus. OFFSET maxRows-1 lands on the last row we keep.
	var threshold uint
	err := r.db.Model(&models.DNSQuery{}).
		Order("id desc").
		Offset(int(maxRows - 1)).
		Limit(1).
		Pluck("id", &threshold).Error
	if err != nil {
		return 0, err
	}
	if threshold == 0 {
		// Fewer than maxRows rows present; nothing to trim.
		return 0, nil
	}
	res := r.db.Where("id < ?", threshold).Delete(&models.DNSQuery{})
	return res.RowsAffected, res.Error
}
