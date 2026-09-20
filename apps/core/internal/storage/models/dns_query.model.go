// SPDX-License-Identifier: GPL-3.0-or-later
package models

import "time"

type DNSQuery struct {
	ID       uint   `gorm:"primaryKey"`
	Domain   string `gorm:"index;not null;"`
	ClientIP string `gorm:"index;not null;"`
	// Action is indexed because the Logs page's Blocked/Allowed pills
	// filter on it, and this table can hold up to ~1M rows on an SD card
	// (see GET /analytics/logs). Added alongside server-side pagination;
	// GORM AutoMigrate adds it in place on an existing table.
	Action          string    `gorm:"index;not null;"` // allow, block, redirect, flagged
	Timestamp       time.Time `gorm:"index;"`
	IsSuspicious    bool      `gorm:"default:false"`
	ThreatScore     float64   `gorm:"default:0"`
	DetectionMethod string    `gorm:"default:''"`
	ThreatReason    string    `gorm:"default:''"`
}

// DetectionMethodDoHBootstrap marks DNSQuery rows produced by the
// DoH/DoT/DoQ bootstrap-hostname interception in
// internal/dnsengine.ProcessDNSQuery (Step 0 — see doh_bootstrap.go).
// Shared here (rather than duplicated as a string literal) because both
// internal/dnsengine (writer) and internal/storage/repositories (reader,
// for GET /analytics/bypass) need the exact same value, and dnsengine
// already imports repositories, so the constant can't live in either of
// those packages without an import cycle.
const DetectionMethodDoHBootstrap = "doh_bootstrap"
