// SPDX-License-Identifier: GPL-3.0-or-later
package models

import "time"

// Token is a per-user API token. The plaintext token value is returned
// exactly once at creation; only the SHA-256 hash is persisted, so losing
// the database does not expose any active bearer token.
type Token struct {
	ID         uint       `gorm:"primaryKey"`
	UserID     uint       `gorm:"not null;index"`
	Hash       string     `gorm:"uniqueIndex;not null"` // hex-encoded SHA-256 of the plaintext token
	Label      string     `gorm:"not null"`             // human-readable, e.g. "laptop-cli", "mcp-gemini"
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	ExpiresAt  *time.Time
}

// Valid reports whether this token can still be used to authenticate at the
// given instant. Expiry and revocation are both expressed as nullable time
// pointers so migrated legacy tokens (no expiry) stay valid indefinitely.
func (t *Token) Valid(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
		return false
	}
	return true
}
