// SPDX-License-Identifier: GPL-3.0-or-later
package repositories

import (
	"errors"
	"strings"
	"time"

	"github.com/lopster568/phantomDNS/internal/storage/models"
	"gorm.io/gorm"
)

// UserRepository owns CRUD on the User model. Password handling stays in
// the handler layer so the repository never touches bcrypt.
type UserRepository interface {
	Create(email, passwordHash, role string) (*models.User, error)
	List() ([]models.User, error)
	Get(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	UpdatePassword(id uint, passwordHash string) error
	UpdateEmail(id uint, email string) error
	UpdateRole(id uint, role string) error
	SetDisabled(id uint, disabled bool) error
	Delete(id uint) error
	TouchLogin(id uint) error
	Count() (int64, error)
}

type gormUserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) UserRepository { return &gormUserRepo{db: db} }

// validRoles is the closed set enforced by the CHECK constraint in the
// User model. Keep the repo layer validation in lockstep so bad input
// produces a clean error instead of a constraint violation.
var validRoles = map[string]bool{
	models.RoleAdmin:    true,
	models.RoleOperator: true,
	models.RoleReadOnly: true,
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *gormUserRepo) Create(email, passwordHash, role string) (*models.User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, errors.New("email is required")
	}
	if passwordHash == "" {
		return nil, errors.New("password hash is required")
	}
	if !validRoles[role] {
		return nil, errors.New("invalid role")
	}
	u := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	}
	if err := r.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (r *gormUserRepo) List() ([]models.User, error) {
	var out []models.User
	err := r.db.Order("created_at ASC").Find(&out).Error
	return out, err
}

func (r *gormUserRepo) Get(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormUserRepo) GetByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("email = ?", normalizeEmail(email)).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormUserRepo) UpdatePassword(id uint, passwordHash string) error {
	if passwordHash == "" {
		return errors.New("password hash is required")
	}
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

func (r *gormUserRepo) UpdateEmail(id uint, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return errors.New("email is required")
	}
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("email", email).Error
}

func (r *gormUserRepo) UpdateRole(id uint, role string) error {
	if !validRoles[role] {
		return errors.New("invalid role")
	}
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("role", role).Error
}

func (r *gormUserRepo) SetDisabled(id uint, disabled bool) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("disabled", disabled).Error
}

func (r *gormUserRepo) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

func (r *gormUserRepo) TouchLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("last_login_at", now).Error
}

func (r *gormUserRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.User{}).Count(&n).Error
	return n, err
}
