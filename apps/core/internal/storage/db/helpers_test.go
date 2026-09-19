// SPDX-License-Identifier: GPL-3.0-or-later
package db

import (
	"testing"
	"time"
)

func timeMustParse(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tt
}

func timePtr(t time.Time) *time.Time { return &t }
