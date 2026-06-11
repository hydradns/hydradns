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
