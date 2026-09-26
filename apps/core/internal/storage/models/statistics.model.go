// SPDX-License-Identifier: Apache-2.0
package models

import "time"

// Statistics will store aggregated DNS query metrics.
type Statistics struct {
	ID                uint   `gorm:"primaryKey"`
	TotalQueries      uint64 `gorm:"default:0"`
	BlockedQueries    uint64 `gorm:"default:0"`
	AllowedQueries    uint64 `gorm:"default:0"`
	RedirectedQueries uint64 `gorm:"default:0"`
	UpdatedAt         time.Time
}
