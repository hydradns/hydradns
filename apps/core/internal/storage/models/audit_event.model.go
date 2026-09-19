// SPDX-License-Identifier: GPL-3.0-or-later
package models

import "time"

// AuditEvent records a single mutating action against the control plane.
// Read operations do not emit audit events - the goal is compliance and
// blame-assignment, not traffic analytics.
//
// ActorID is nullable so system-initiated events (migrations, scheduled
// jobs) can be recorded without a user attached.
//
// BeforeJSON / AfterJSON carry the target object's state as serialized
// JSON blobs. Keeping them as TEXT lets handlers evolve their models
// without schema churn - the tradeoff is queries must parse JSON to
// filter on nested fields, which is fine for compliance workloads.
type AuditEvent struct {
	ID         uint      `gorm:"primaryKey"`
	ActorID    *uint     `gorm:"index"`
	Action     string    `gorm:"not null;index"` // e.g. "policy.create", "blocklist.delete"
	Target     string    `gorm:"not null"`       // e.g. "policy:block-social"
	BeforeJSON *string   // nullable; absent on creates
	AfterJSON  *string   // nullable; absent on deletes
	ClientIP   string    `gorm:"not null"`
	UserAgent  string    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index"`
}
