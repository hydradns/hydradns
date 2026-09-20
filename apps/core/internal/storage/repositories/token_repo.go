// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"gorm.io/gorm"
)

// DefaultTokenExpiry controls how long newly-minted tokens remain valid
// by default. Callers can override per-token via CreateForUser.
const DefaultTokenExpiry = 90 * 24 * time.Hour

// TokenRepository owns lifecycle of API tokens. Tokens are stored as
// SHA-256 hashes; the plaintext is only ever returned at creation time.
type TokenRepository interface {
	// CreateForUser mints a new token for the given user, returns the
	// plaintext (caller must surface to the user exactly once) and the
	// persisted Token row. expiry <= 0 means use DefaultTokenExpiry; pass
	// a sentinel to opt out entirely via CreateUnexpiring.
	CreateForUser(userID uint, label string, expiry time.Duration) (plaintext string, tok *models.Token, err error)

	// CreateUnexpiring mints a token with no expiry (ExpiresAt=NULL).
	// Used by the legacy-singleton migration so pre-RBAC sessions keep
	// working through the upgrade.
	CreateUnexpiring(userID uint, label string) (plaintext string, tok *models.Token, err error)

	// ResolveActive looks up a token by plaintext, verifies it is not
	// revoked or expired, and bumps LastUsedAt. Returns the token row
	// and the owning user. Returns (nil, nil, nil) on miss/invalid so
	// the caller can respond 401 without distinguishing cause.
	ResolveActive(plaintext string) (*models.Token, *models.User, error)

	// ListForUser returns the user's tokens without plaintext.
	ListForUser(userID uint) ([]models.Token, error)

	// Revoke marks the token as revoked. Idempotent.
	Revoke(id uint) error
}

type gormTokenRepo struct{ db *gorm.DB }

func NewTokenRepo(db *gorm.DB) TokenRepository { return &gormTokenRepo{db: db} }

// generatePlaintext returns a 32-byte random token encoded as 64 hex chars.
// Cryptographically random source. Exported for tests.
func generatePlaintext() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashToken is an alias for models.HashToken, re-exported here so callers
// already holding a repositories import do not need to pull in models
// just to hash a token. Both point at the same implementation.
func HashToken(plaintext string) string { return models.HashToken(plaintext) }

func (r *gormTokenRepo) CreateForUser(userID uint, label string, expiry time.Duration) (string, *models.Token, error) {
	if expiry <= 0 {
		expiry = DefaultTokenExpiry
	}
	expAt := time.Now().Add(expiry)
	return r.create(userID, label, &expAt)
}

func (r *gormTokenRepo) CreateUnexpiring(userID uint, label string) (string, *models.Token, error) {
	return r.create(userID, label, nil)
}

func (r *gormTokenRepo) create(userID uint, label string, expiresAt *time.Time) (string, *models.Token, error) {
	if label == "" {
		return "", nil, errors.New("token label is required")
	}
	plaintext, err := generatePlaintext()
	if err != nil {
		return "", nil, err
	}
	tok := &models.Token{
		UserID:    userID,
		Hash:      models.HashToken(plaintext),
		Label:     label,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := r.db.Create(tok).Error; err != nil {
		return "", nil, err
	}
	return plaintext, tok, nil
}

func (r *gormTokenRepo) ResolveActive(plaintext string) (*models.Token, *models.User, error) {
	if plaintext == "" {
		return nil, nil, nil
	}
	hash := models.HashToken(plaintext)

	var tok models.Token
	err := r.db.Where("hash = ?", hash).First(&tok).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if !tok.Valid(time.Now()) {
		return nil, nil, nil
	}

	var user models.User
	if err := r.db.First(&user, tok.UserID).Error; err != nil {
		// Token row outlived its user (should not happen absent manual
		// SQL); treat as unauthenticated, do not 500 on the caller.
		return nil, nil, nil
	}
	if user.Disabled {
		return nil, nil, nil
	}

	// Bump LastUsedAt. Best-effort; a failure here is observational.
	now := time.Now()
	r.db.Model(&models.Token{}).Where("id = ?", tok.ID).Update("last_used_at", now)
	tok.LastUsedAt = &now

	return &tok, &user, nil
}

func (r *gormTokenRepo) ListForUser(userID uint) ([]models.Token, error) {
	var out []models.Token
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *gormTokenRepo) Revoke(id uint) error {
	now := time.Now()
	return r.db.Model(&models.Token{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}
