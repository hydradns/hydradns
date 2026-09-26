// SPDX-License-Identifier: Apache-2.0
package models

import "time"

// Statistics will store aggregated DNS query metrics.
type SystemState struct {
	ID            uint `gorm:"primaryKey"`
	DNSEnabled    bool `gorm:"not null"`
	PolicyEnabled bool `gorm:"not null"`
	// UpdatedBy string
	UpdatedAt time.Time
}
