// SPDX-License-Identifier: Apache-2.0
package db

import (
	"log"
	"os"
	"path/filepath"
)

// DefaultDBPath is where the SQLite database lives when HYDRA_DB is not
// set. Both cmd/controlplane and cmd/dataplane resolve their DB path
// through ResolveDBPath so this stays the single source of truth.
const DefaultDBPath = "/app/data/hydradns.db"

// legacyDBFileName was this file's name before the PhantomDNS -> HydraDNS
// rename. Kept only so ResolveDBPath can fall back to it.
const legacyDBFileName = "phantomdns.db"

// ResolveDBPath decides which sqlite file to open. path is normally
// os.Getenv("HYDRA_DB"); an empty string means "use DefaultDBPath."
//
// If the resolved path doesn't exist yet but a legacy phantomdns.db sits
// in the same directory, that legacy file is returned instead: otherwise
// an in-place upgrade would silently start a brand-new, empty hydradns.db
// next to an existing install's real data. This covers both:
//   - no HYDRA_DB set: DefaultDBPath doesn't exist yet, phantomdns.db does.
//   - HYDRA_DB explicitly set to a hydradns.db path (as docker-compose.yml
//     does): same situation, just with an explicit path instead of the
//     default.
//
// Logs one info line when the fallback triggers; otherwise silent.
func ResolveDBPath(path string) string {
	if path == "" {
		path = DefaultDBPath
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}

	legacy := filepath.Join(filepath.Dir(path), legacyDBFileName)
	if _, err := os.Stat(legacy); err == nil {
		log.Printf("%s not found; using existing legacy database %s instead (pre-HydraDNS-rename filename)", path, legacy)
		return legacy
	}

	// Neither exists: fresh install. InitDB will create path.
	return path
}
