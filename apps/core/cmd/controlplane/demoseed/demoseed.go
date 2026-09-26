// SPDX-License-Identifier: Apache-2.0

// Package demoseed provides the startup data (and periodic refresh) for a
// public, read-only demo deployment (HYDRA_DEMO_MODE=true). It is only
// ever invoked from cmd/controlplane/main.go when that flag is set, and
// touches nothing when it isn't. The whole point of keeping this in its
// own package is that it can be deleted (this file, its test, and the two
// call sites in main.go) without touching anything else in the control
// plane.
//
// It does three things:
//  1. EnsureDemoUser creates a fixed-password, role=read_only "demo" user
//     idempotently (see cmd/controlplane/middlewares.DemoGuard for the
//     actual write-blocking boundary; this package never creates an
//     account with write access).
//  2. SeedIfEmpty fills an empty database with deterministic, synthetic
//     data (policies, blocklist sources + entries, ~7 days of query logs,
//     matching statistics) so every chart on the dashboard has something
//     to show instead of a blank state.
//  3. Refresh / StartRefreshLoop periodically re-anchors the query-log
//     timestamps to "now" so a long-running demo container doesn't drift
//     into showing a stale "last activity: 3 weeks ago" dashboard.
package demoseed

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// DemoUserEmail / DemoUserPassword are the fixed, publicly documented
// demo credentials (see demo/README.md). They are intentionally not
// secret; the security boundary for a public demo is middlewares.
// DemoGuard (every mutating request is rejected before it reaches auth),
// not the obscurity of this password.
const (
	DemoUserEmail    = "demo@hydradns.local"
	DemoUserPassword = "hydradns-demo"
)

// seedRandSource is a fixed seed so the generated dataset is deterministic
// across restarts and redeployments (same domains, same rough shape, same
// number of rows every time), useful for anyone taking screenshots of, or
// writing docs against, the public demo.
const seedRandSource = 42

// EnsureDemoUser idempotently creates the read_only demo user. It refuses
// to run (returns an error, which main.go treats as fatal) if the users
// table already contains anything other than exactly that one demo
// account; see the safety rationale below.
//
// Safety rationale: HYDRA_DEMO_MODE is meant to run against a dedicated,
// disposable volume (see demo/docker-compose.demo.yml). If it were ever
// set against a real deployment's database by mistake, blindly proceeding
// would (a) add a publicly-documented-password backdoor account to a real
// instance, and (b) hand seeded synthetic query-log rows to Refresh, which
// periodically deletes and re-inserts the entire dns_queries table,
// destroying real query history. Refusing to start is the safe failure
// mode; the fix is "use a fresh volume", which the shipped demo compose
// file already does.
func EnsureDemoUser(store *repositories.Store) error {
	n, err := store.Users.Count()
	if err != nil {
		return fmt.Errorf("demoseed: counting users: %w", err)
	}

	if n == 0 {
		// A DB with query-log history but no users at all is not a fresh
		// demo volume, even though the "zero users" check alone would let
		// it through: it's a pre-RBAC volume, or one where migration
		// hasn't run yet. Proceeding would seed the demo user over it, and
		// the periodic Refresh loop would then delete that real history on
		// its very first tick.
		if queryCount, qerr := store.QueryLogs.Count(); qerr != nil {
			return fmt.Errorf("demoseed: counting query logs: %w", qerr)
		} else if queryCount > 0 {
			return fmt.Errorf("demoseed: HYDRA_DEMO_MODE=true but the database has %d query log row(s) and no users — "+
				"refusing to seed a public demo (whose periodic refresh deletes and regenerates dns_queries) over what "+
				"looks like real history; use a fresh volume for the demo (see demo/docker-compose.demo.yml)", queryCount)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(DemoUserPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("demoseed: hashing demo password: %w", err)
		}
		if _, err := store.Users.Create(DemoUserEmail, string(hash), models.RoleReadOnly); err != nil {
			return fmt.Errorf("demoseed: creating demo user: %w", err)
		}
		log.Printf("demoseed: created demo user %q (role=%s)", DemoUserEmail, models.RoleReadOnly)
		return nil
	}

	// A user already exists. That's only acceptable if it is exactly the
	// demo account restarting against its own (already-seeded) volume.
	existing, err := store.Users.GetByEmail(DemoUserEmail)
	if err != nil || existing == nil {
		return errors.New("demoseed: HYDRA_DEMO_MODE=true but the database already has non-demo user(s) — " +
			"refusing to seed a public demo over what looks like real data; use a fresh volume for the demo " +
			"(see demo/docker-compose.demo.yml)")
	}
	if n > 1 {
		return fmt.Errorf("demoseed: HYDRA_DEMO_MODE=true but the database has %d users, not just the demo account — "+
			"refusing to seed a public demo over what looks like real data; use a fresh volume for the demo", n)
	}
	if existing.Role != models.RoleReadOnly {
		return fmt.Errorf("demoseed: the existing %q user has role %q, not %q — refusing to run demo mode against it",
			DemoUserEmail, existing.Role, models.RoleReadOnly)
	}
	// Exactly the demo account, already present (a restart); nothing to do.
	return nil
}

// SeedIfEmpty populates a fresh database with synthetic demo data, but
// only if the query log is empty, so it never duplicates data on a plain
// container restart against an already-seeded volume. Call Refresh
// instead (see below) for the periodic re-anchor.
func SeedIfEmpty(store *repositories.Store) error {
	n, err := store.QueryLogs.Count()
	if err != nil {
		return fmt.Errorf("demoseed: counting query logs: %w", err)
	}
	if n > 0 {
		return nil
	}
	return seedAll(store, time.Now())
}

// Refresh re-anchors the demo dataset to "now": it deletes the existing
// query-log rows and statistics counters and regenerates them with the
// same deterministic generator, just anchored at the current time instead
// of whenever the container first booted. Policies and blocklist sources
// are left alone (seedAll only creates those when they don't already
// exist, and after the first Seed/Refresh they do); they have no
// "recency" requirement the way the logs/charts do.
//
// This is the "periodic light refresh" approach rather than incrementally
// shifting every row's timestamp forward: regenerating from the same seed
// is simpler to get right (no per-row arithmetic across SQLite's several
// on-disk datetime formats, see repositories.parseSQLiteTime) and produces
// an identical-shaped dataset every time.
//
// The whole thing runs inside one transaction: the delete, the statistics
// reset, and the batched re-insert must land together, or a dashboard
// request landing between them could see an empty or half-populated
// table, charts dropping to zero mid-demo, for no reason a viewer could
// tell apart from a real outage. WAL readers see either the pre-refresh or
// post-refresh state, never a gap, and the batched inserts inside
// seedQueryLogs (CreateInBatches, 200 rows at a time) keep the transaction
// itself short: a few thousand rows is comfortably sub-second on SQLite.
func Refresh(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.DNSQuery{}).Error; err != nil {
			return fmt.Errorf("demoseed: clearing query logs for refresh: %w", err)
		}
		if err := tx.Model(&models.Statistics{}).Where("id = ?", 1).Updates(map[string]interface{}{
			"total_queries":      0,
			"blocked_queries":    0,
			"allowed_queries":    0,
			"redirected_queries": 0,
			"updated_at":         time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("demoseed: resetting statistics for refresh: %w", err)
		}
		// seedAll (specifically seedQueryLogs) must run against tx, not the
		// outer db, so its batched inserts join the same transaction as
		// the delete and the statistics reset above. A tx-scoped store is
		// built here (repositories only hold a *gorm.DB handle, and
		// NewStore(tx) makes every repository method issue its queries
		// against tx instead of the connection pool).
		return seedAll(repositories.NewStore(tx), time.Now())
	})
}

// StartRefreshLoop runs Refresh on a ticker until stop is closed. Intended
// to be launched with `go demoseed.StartRefreshLoop(...)` from main.go
// only when HYDRA_DEMO_MODE=true. Errors are logged, not fatal: a failed
// refresh leaves the previous (still valid, just older) dataset in place
// rather than crashing a running public demo.
func StartRefreshLoop(db *gorm.DB, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := Refresh(db); err != nil {
				log.Printf("demoseed: periodic refresh failed: %v", err)
			}
		case <-stop:
			return
		}
	}
}

// --- generation ---

var allowedDomains = []string{
	"google.com", "youtube.com", "github.com", "wikipedia.org", "amazon.com",
	"netflix.com", "microsoft.com", "apple.com", "cloudflare.com", "reddit.com",
	"stackoverflow.com", "nytimes.com", "spotify.com", "zoom.us", "slack.com",
}

var blockedAdDomains = []string{
	"doubleclick.net", "ads.example.com", "adnetwork.example.com",
	"tracker.example.com", "adservice.example.com",
}

var blockedMalwareDomains = []string{
	"malware-test.com", "badexample.com", "phishing-example.net", "c2-example.org",
}

// dohBootstrapDomains mirrors the well-known DoH/DoT/DoQ provider
// hostnames called out in docs/pi-deployment.md's "block DNS-over-HTTPS"
// example policy, so the seeded bypass-attempts panel tells a consistent
// story with the seeded "block-doh" policy below.
var dohBootstrapDomains = []string{
	"dns.google", "cloudflare-dns.com", "doh.opendns.com", "dns.quad9.net",
}

var suspiciousDomains = []string{
	"xj3kd9fbaq.info", "qpzmvr8s7n.top", "kf92hslqzt.biz", "a8f0e2b1c9.click",
	"m4x9z1q7wp.xyz",
}

// clientIPPool is entirely RFC 1918 (private) address space: a handful of
// "home LAN" and "office LAN" style ranges so the seeded data reads as a
// small fleet of client devices rather than one host.
var clientIPPool = []string{
	"192.168.1.10", "192.168.1.11", "192.168.1.23", "192.168.1.42", "192.168.1.87", "192.168.1.105",
	"10.0.0.5", "10.0.0.6", "10.0.0.42", "10.0.1.15",
	"172.16.0.10", "172.16.0.11", "172.16.5.20",
}

func mustJSON(v []string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// seedPolicies creates a small, illustrative policy set if none exist yet.
func seedPolicies(store *repositories.Store) error {
	existing, err := store.Policies.List()
	if err != nil {
		return fmt.Errorf("listing policies: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}

	now := time.Now()
	policies := []*models.Policy{
		{
			ID: "demo-block-ads", Name: "Block Ads", Description: "Blocks known ad and tracking domains.",
			Category: "advertising", Action: "BLOCK", Priority: 100, Enabled: true,
			Domains: mustJSON(blockedAdDomains), CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "demo-block-malware", Name: "Block Malware", Description: "Blocks known malware and phishing domains.",
			Category: "security", Action: "BLOCK", Priority: 200, Enabled: true,
			Domains: mustJSON(blockedMalwareDomains), CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "demo-block-doh", Name: "Block DNS-over-HTTPS", Description: "Blocks well-known DoH/DoT/DoQ bootstrap hosts used to bypass DNS filtering.",
			Category: "security", Action: "BLOCK", Priority: 150, Enabled: true,
			Domains: mustJSON(dohBootstrapDomains), CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "demo-allow-corporate-vpn", Name: "Allow Corporate VPN", Description: "Always allows the corporate VPN endpoint.",
			Category: "business", Action: "ALLOW", Priority: 50, Enabled: true,
			Domains: mustJSON([]string{"vpn.corp-example.com"}), CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "demo-redirect-safe-mode", Name: "Redirect Safe Mode", Description: "Redirects a legacy hostname to the safe-mode landing page.",
			Category: "compliance", Action: "REDIRECT", RedirectIP: "192.168.1.1", Priority: 75, Enabled: true,
			Domains: mustJSON([]string{"safe-mode.example"}), CreatedAt: now, UpdatedAt: now,
		},
	}
	for _, p := range policies {
		if err := store.Policies.Create(p); err != nil {
			return fmt.Errorf("creating policy %s: %w", p.ID, err)
		}
	}
	return nil
}

// blocklistSeed describes one synthetic blocklist source: its metadata
// plus how many synthetic filler domains to generate alongside a handful
// of "real" (already-referenced-in-logs) domains.
type blocklistSeed struct {
	id, name, category string
	realDomains        []string
	fillerPrefix       string
	fillerCount        int
}

// seedBlocklists creates a few blocklist sources with real snapshot/entry
// rows (so GET /blocklists reports non-zero domains_count) if none exist
// yet. No network fetch happens: entries are generated locally, which is
// also why this is safe to run with the dataplane's DNS ports unpublished.
func seedBlocklists(store *repositories.Store) error {
	existing, err := store.Blocklist.ListSources()
	if err != nil {
		return fmt.Errorf("listing blocklist sources: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}

	seeds := []blocklistSeed{
		{
			id: "demo-block-ads", name: "Demo Ad Block List", category: "advertising",
			realDomains: blockedAdDomains, fillerPrefix: "adnetwork-", fillerCount: 800,
		},
		{
			id: "demo-block-malware", name: "Demo Malware Block List", category: "security",
			realDomains: blockedMalwareDomains, fillerPrefix: "malicious-", fillerCount: 300,
		},
		{
			id: "demo-block-doh", name: "Demo DoH/DoT/DoQ Bootstrap List", category: "security",
			realDomains: dohBootstrapDomains, fillerPrefix: "", fillerCount: 0,
		},
	}

	now := time.Now()
	for _, s := range seeds {
		src := &models.BlocklistSource{
			ID: s.id, Name: s.name, URL: "local://demoseed/" + s.id, Format: "domain-list",
			Category: s.category, Enabled: true, CreatedAt: now,
		}
		if err := store.Blocklist.CreateSource(src); err != nil {
			return fmt.Errorf("creating blocklist source %s: %w", s.id, err)
		}

		entries := make([]models.BlocklistEntry, 0, len(s.realDomains)+s.fillerCount)
		for _, d := range s.realDomains {
			entries = append(entries, models.BlocklistEntry{Domain: d, SourceID: s.id, Category: s.category, CreatedAt: now, UpdatedAt: now})
		}
		for i := 0; i < s.fillerCount; i++ {
			d := fmt.Sprintf("%s%d.example.net", s.fillerPrefix, i)
			entries = append(entries, models.BlocklistEntry{Domain: d, SourceID: s.id, Category: s.category, CreatedAt: now, UpdatedAt: now})
		}

		checksum := fmt.Sprintf("demoseed-%s-%d", s.id, len(entries))
		if _, err := store.Blocklist.SaveSnapshotWithEntries(*src, checksum, entries); err != nil {
			return fmt.Errorf("saving snapshot for %s: %w", s.id, err)
		}
	}
	return nil
}

// totalDemoQueryRows is the number of synthetic DNSQuery rows generated
// per seed/refresh cycle, spread uniformly over the trailing 7 days from
// the anchor time. Large enough that the logs page's pagination and the
// overview charts have real shape; small enough that startup/refresh
// stays fast (a few thousand rows batch-inserted is well under a second
// on SQLite).
const totalDemoQueryRows = 3000

// seedQueryLogs generates totalDemoQueryRows synthetic DNSQuery rows
// spread over the 7 days before anchor, and updates the statistics
// singleton to match. The action/detection mix is deliberately weighted
// so every dashboard chart (allow vs block rate, suspicious flags, the
// bypass-attempts panel) has something to show:
//   - ~72% allow (well-known domains)
//   - ~15% block (ad domains)
//   - ~5%  block (malware domains)
//   - ~3%  redirect
//   - ~3%  flagged + suspicious (DGA-like domains)
//   - ~2%  block + suspicious, detection_method=doh_bootstrap
func seedQueryLogs(store *repositories.Store, anchor time.Time) error {
	rng := rand.New(rand.NewSource(seedRandSource))

	rows := make([]*models.DNSQuery, 0, totalDemoQueryRows)
	var allowed, blocked, redirected, other int64

	window := 7 * 24 * time.Hour
	for i := 0; i < totalDemoQueryRows; i++ {
		ts := anchor.Add(-time.Duration(rng.Int63n(int64(window))))
		clientIP := clientIPPool[rng.Intn(len(clientIPPool))]
		roll := rng.Float64()

		q := &models.DNSQuery{ClientIP: clientIP, Timestamp: ts}

		switch {
		case roll < 0.02: // DoH/DoT/DoQ bootstrap interception
			q.Domain = dohBootstrapDomains[rng.Intn(len(dohBootstrapDomains))]
			q.Action = "block"
			q.IsSuspicious = true
			q.ThreatScore = 0.85 + rng.Float64()*0.1
			q.DetectionMethod = models.DetectionMethodDoHBootstrap
			q.ThreatReason = "DNS-over-HTTPS bootstrap hostname intercepted"
			blocked++
		case roll < 0.05: // suspicious DGA-like domain
			q.Domain = suspiciousDomains[rng.Intn(len(suspiciousDomains))]
			q.Action = "flagged"
			q.IsSuspicious = true
			q.ThreatScore = 0.6 + rng.Float64()*0.3
			q.ThreatReason = "high-entropy subdomain (possible DGA)"
			other++
		case roll < 0.20: // ad blocking
			q.Domain = blockedAdDomains[rng.Intn(len(blockedAdDomains))]
			q.Action = "block"
			blocked++
		case roll < 0.25: // malware blocking
			q.Domain = blockedMalwareDomains[rng.Intn(len(blockedMalwareDomains))]
			q.Action = "block"
			blocked++
		case roll < 0.28: // redirect
			q.Domain = "safe-mode.example"
			q.Action = "redirect"
			redirected++
		default: // allow
			q.Domain = allowedDomains[rng.Intn(len(allowedDomains))]
			q.Action = "allow"
			allowed++
		}

		rows = append(rows, q)
	}

	if err := store.QueryLogs.SaveBatch(rows); err != nil {
		return fmt.Errorf("saving synthetic query logs: %w", err)
	}
	if err := store.Statistics.AddCounters(allowed, blocked, redirected, other); err != nil {
		return fmt.Errorf("updating statistics: %w", err)
	}
	return nil
}

func seedAll(store *repositories.Store, anchor time.Time) error {
	if err := seedPolicies(store); err != nil {
		return fmt.Errorf("demoseed: %w", err)
	}
	if err := seedBlocklists(store); err != nil {
		return fmt.Errorf("demoseed: %w", err)
	}
	if err := seedQueryLogs(store, anchor); err != nil {
		return fmt.Errorf("demoseed: %w", err)
	}
	log.Printf("demoseed: seeded %d synthetic query log rows anchored at %s", totalDemoQueryRows, anchor.Format(time.RFC3339))
	return nil
}
