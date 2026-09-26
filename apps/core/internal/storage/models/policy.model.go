// SPDX-License-Identifier: Apache-2.0
package models

import "time"

type Policy struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"not null"`
	Description string
	Category    string `gorm:"index"`
	Action      string `gorm:"not null"` // BLOCK, ALLOW, REDIRECT
	RedirectIP  string
	Domains     string `gorm:"type:text"` // JSON array stored as text
	Priority    int    `gorm:"default:0"`
	Enabled     bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
