// SPDX-License-Identifier: GPL-3.0-or-later
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDBPath_EmptyEnvUsesDefault(t *testing.T) {
	got := ResolveDBPath("")
	if got != DefaultDBPath {
		t.Errorf("ResolveDBPath(\"\") = %q, want default %q", got, DefaultDBPath)
	}
}

func TestResolveDBPath_ExplicitPathUsedWhenFileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydradns.db")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := ResolveDBPath(target)
	if got != target {
		t.Errorf("ResolveDBPath(%q) = %q, want unchanged %q (file already exists)", target, got, target)
	}
}

func TestResolveDBPath_FallsBackToLegacyFileWhenNewOneMissing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydradns.db")
	legacy := filepath.Join(dir, "phantomdns.db")
	if err := os.WriteFile(legacy, []byte("legacy-data"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := ResolveDBPath(target)
	if got != legacy {
		t.Errorf("ResolveDBPath(%q) = %q, want fallback to legacy %q", target, got, legacy)
	}
}

func TestResolveDBPath_NeitherFileExistsUsesRequestedPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydradns.db")

	got := ResolveDBPath(target)
	if got != target {
		t.Errorf("ResolveDBPath(%q) = %q, want unchanged %q (fresh install, no legacy file)", target, got, target)
	}
}

// TestResolveDBPath_ExplicitEnvPointingAtMissingHydraDNSStillFindsLegacy
// covers docker-compose.yml explicitly setting HYDRA_DB=/app/data/hydradns.db:
// on an existing install upgrading in place, that file won't exist yet, but
// phantomdns.db will still be sitting in the same directory.
func TestResolveDBPath_ExplicitEnvPointingAtMissingHydraDNSStillFindsLegacy(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydradns.db") // as if HYDRA_DB=.../hydradns.db
	legacy := filepath.Join(dir, "phantomdns.db")
	if err := os.WriteFile(legacy, []byte("legacy-data"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got := ResolveDBPath(target)
	if got != legacy {
		t.Errorf("ResolveDBPath(%q) = %q, want fallback to legacy %q", target, got, legacy)
	}
}
