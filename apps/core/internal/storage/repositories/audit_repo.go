// SPDX-License-Identifier: Apache-2.0
package repositories

import (
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"gorm.io/gorm"
)

// AuditFilter narrows a Query call. Zero-value fields are ignored; only
// the fields that are set get applied as AND-joined predicates.
type AuditFilter struct {
	ActorID *uint
	Action  string
	Target  string
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

// AuditRepository persists and queries audit events. Writes are
// best-effort: the caller should log failures but must not fail the
// originating request. See audit.Record in the handlers layer.
type AuditRepository interface {
	Record(evt *models.AuditEvent) error
	Query(filter AuditFilter) ([]models.AuditEvent, error)
	Count(filter AuditFilter) (int64, error)
}

type gormAuditRepo struct{ db *gorm.DB }

func NewAuditRepo(db *gorm.DB) AuditRepository { return &gormAuditRepo{db: db} }

func (r *gormAuditRepo) Record(evt *models.AuditEvent) error {
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now()
	}
	return r.db.Create(evt).Error
}

func (r *gormAuditRepo) Query(filter AuditFilter) ([]models.AuditEvent, error) {
	q := r.applyFilter(r.db.Model(&models.AuditEvent{}), filter).Order("created_at DESC")
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	var out []models.AuditEvent
	err := q.Find(&out).Error
	return out, err
}

func (r *gormAuditRepo) Count(filter AuditFilter) (int64, error) {
	var n int64
	err := r.applyFilter(r.db.Model(&models.AuditEvent{}), filter).Count(&n).Error
	return n, err
}

func (r *gormAuditRepo) applyFilter(q *gorm.DB, filter AuditFilter) *gorm.DB {
	if filter.ActorID != nil {
		q = q.Where("actor_id = ?", *filter.ActorID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.Target != "" {
		q = q.Where("target = ?", filter.Target)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}
	return q
}
