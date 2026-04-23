// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"time"

	"github.com/lopster568/phantomDNS/internal/logger"
	"github.com/lopster568/phantomDNS/internal/storage/models"
	"gorm.io/gorm"
)

// Interface
type StatisticsRepository interface {
	Save(stat *models.Statistics) error
	ListRecent(limit int) ([]models.Statistics, error)
	IncrementCounter(action string) error
	SeedSingleton() error
}

// Implementation
type GormStatisticsRepo struct {
	db *gorm.DB
}

func NewGormStatisticsRepo(db *gorm.DB) *GormStatisticsRepo {
	return &GormStatisticsRepo{db: db}
}

func (r *GormStatisticsRepo) Save(stat *models.Statistics) error {
	stat.UpdatedAt = time.Now()
	logger.Log.Debug("Saving statistics record")
	logger.Log.Debug("stats", stat)
	return r.db.Save(stat).Error
}

func (r *GormStatisticsRepo) ListRecent(limit int) ([]models.Statistics, error) {
	var stats []models.Statistics
	err := r.db.Order("updated_at desc").Limit(limit).Find(&stats).Error
	return stats, err
}

// IncrementCounter increments the global counters (single-row statistics).
//
// Uses a single atomic UPDATE so concurrent DNS query handlers cannot
// race each other into lost updates (or, on cold boot, duplicate INSERTs
// colliding on id=1). The singleton row is seeded by SeedSingleton at
// startup, so the runtime path is update-only.
func (r *GormStatisticsRepo) IncrementCounter(action string) error {
	var bumpCol string
	switch action {
	case "allow":
		bumpCol = "allowed_queries"
	case "block":
		bumpCol = "blocked_queries"
	case "redirect":
		bumpCol = "redirected_queries"
	}

	updates := map[string]interface{}{
		"total_queries": gorm.Expr("total_queries + 1"),
		"updated_at":    time.Now(),
	}
	if bumpCol != "" {
		updates[bumpCol] = gorm.Expr(bumpCol + " + 1")
	}

	res := r.db.Model(&models.Statistics{}).Where("id = ?", 1).Updates(updates)
	if res.Error != nil {
		logger.Log.Error("Failed to increment statistics: " + res.Error.Error())
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Seed row is missing (first boot before SeedSingleton, or a
		// volume wipe between binary restarts). Recreate it idempotently.
		if err := r.SeedSingleton(); err != nil {
			return err
		}
		// Retry the update once. If it still reports 0 rows, something
		// else is wrong and the error path above will surface it on the
		// next query.
		r.db.Model(&models.Statistics{}).Where("id = ?", 1).Updates(updates)
	}
	return nil
}

// SeedSingleton inserts the statistics row with id=1 if it does not exist.
// Safe to call many times; the INSERT is guarded by ON CONFLICT DO NOTHING
// semantics via the unique primary key.
func (r *GormStatisticsRepo) SeedSingleton() error {
	// Raw SQL keeps the insert idempotent across drivers without
	// pulling in clause.OnConflict, which behaves differently on SQLite.
	return r.db.Exec(
		"INSERT OR IGNORE INTO statistics (id, total_queries, blocked_queries, allowed_queries, redirected_queries, updated_at) VALUES (1, 0, 0, 0, 0, ?)",
		time.Now(),
	).Error
}
