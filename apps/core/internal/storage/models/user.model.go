// SPDX-License-Identifier: Apache-2.0
package models

import "time"

// Role names. Keep as string constants so GORM's check constraint and
// the middleware layer share the same source of truth.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleReadOnly = "read_only"
)

// User is an authenticated principal for the control plane.
// Replaces the singleton AdminCredential. See docs/phase6-rbac-plan.md.
type User struct {
	ID           uint       `gorm:"primaryKey"`
	Email        string     `gorm:"uniqueIndex;not null"`
	PasswordHash string     `gorm:"not null"`
	Role         string     `gorm:"not null;check:role IN ('admin','operator','read_only')"`
	Disabled     bool       `gorm:"not null;default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}
