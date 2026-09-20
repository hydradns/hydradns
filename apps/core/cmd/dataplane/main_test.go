// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"testing"
	"time"

	"github.com/hydradns/hydra-core/internal/blocklist"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
)

// fakeBlocklistRepo is a minimal repositories.BlocklistRepository stub for
// exercising refreshSources without a real DB. Only ListSources is
// exercised by the tests below (zero/erroring source lists); the rest are
// no-op stubs required to satisfy the interface.
type fakeBlocklistRepo struct {
	sources    []models.BlocklistSource
	listSrcErr error
}

func (f *fakeBlocklistRepo) SaveSnapshotWithEntries(src models.BlocklistSource, checksum string, entries []models.BlocklistEntry) (models.BlocklistSnapshot, error) {
	return models.BlocklistSnapshot{}, nil
}
func (f *fakeBlocklistRepo) GetAll() ([]string, error)             { return nil, nil }
func (f *fakeBlocklistRepo) GetAllEnabled() ([]string, error)      { return nil, nil }
func (f *fakeBlocklistRepo) IsBlocked(domain string) (bool, error) { return false, nil }
func (f *fakeBlocklistRepo) ListSources() ([]models.BlocklistSource, error) {
	return f.sources, f.listSrcErr
}
func (f *fakeBlocklistRepo) GetSource(id string) (*models.BlocklistSource, error) { return nil, nil }
func (f *fakeBlocklistRepo) CreateSource(src *models.BlocklistSource) error       { return nil }
func (f *fakeBlocklistRepo) UpdateSourceFields(src *models.BlocklistSource) error { return nil }
func (f *fakeBlocklistRepo) DeleteSource(id string) error                         { return nil }
func (f *fakeBlocklistRepo) CountEntriesBySource(sourceID string) (int64, error)  { return 0, nil }
func (f *fakeBlocklistRepo) CountEntriesGroupedBySource() (map[string]int64, error) {
	return nil, nil
}
func (f *fakeBlocklistRepo) Signature() (repositories.BlocklistSignature, error) {
	return repositories.BlocklistSignature{}, nil
}

// TestRefreshSources_ForcesRebuildWithNoSources verifies the 6h
// refreshSources pass forces an in-memory rebuild regardless of
// signature, even when there are no sources configured: the signature-
// based poll loop alone would skip a rebuild here since nothing changed.
func TestRefreshSources_ForcesRebuildWithNoSources(t *testing.T) {
	repo := &fakeBlocklistRepo{}
	engine := blocklist.NewEngine(repo)

	fakeSrc := &fakeBlocklistSource{sig: repositories.BlocklistSignature{SourceCount: 1}}
	r := newBlocklistReloader(fakeSrc, blocklist.NewMemoryChecker())

	// Baseline load; a plain Poll() afterwards would no-op (unchanged sig).
	r.Poll()
	waitUntil(t, time.Second, func() bool { return fakeSrc.calls() == 1 })

	refreshSources(context.Background(), engine, r)

	waitUntil(t, time.Second, func() bool { return fakeSrc.calls() == 2 })
}

// Same safety net must still fire even when listing sources itself errors:
// refreshSources must not skip the forced rebuild just because it couldn't
// refresh anything this pass.
func TestRefreshSources_ForcesRebuildOnListSourcesError(t *testing.T) {
	repo := &fakeBlocklistRepo{listSrcErr: errTest}
	engine := blocklist.NewEngine(repo)

	fakeSrc := &fakeBlocklistSource{sig: repositories.BlocklistSignature{SourceCount: 1}}
	r := newBlocklistReloader(fakeSrc, blocklist.NewMemoryChecker())

	r.Poll()
	waitUntil(t, time.Second, func() bool { return fakeSrc.calls() == 1 })

	refreshSources(context.Background(), engine, r)

	waitUntil(t, time.Second, func() bool { return fakeSrc.calls() == 2 })
}
