package main

// SPDX-License-Identifier: GPL-3.0-or-later
import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hydradns/hydra-core/internal/blocklist"
	"github.com/hydradns/hydra-core/internal/config"
	"github.com/hydradns/hydra-core/internal/dnsengine"
	dataplanegrpc "github.com/hydradns/hydra-core/internal/grpc/dataplane"
	"github.com/hydradns/hydra-core/internal/logger"
	"github.com/hydradns/hydra-core/internal/policy"
	"github.com/hydradns/hydra-core/internal/storage/db"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
	"github.com/hydradns/hydra-core/internal/utils"
)

func main() {
	logger.Log.Info("Starting HydraDNS Data Plane...")

	// 1. Initialize DB
	dbPath := db.ResolveDBPath(os.Getenv("HYDRA_DB"))
	db.InitDB(dbPath)

	// 1b. Anonymization secret — HMAC key AnonymizeIP uses to hash client
	// IPs before they're written to the query log, IF anonymization is
	// enabled (it's off by default: per-device visibility in the query log
	// is a core feature, so this is opt-in). Resolving/generating a secret
	// nobody will use is harmless but surprising, so skip it entirely when
	// disabled. Must run, and InitSecret must be called, before the DNS
	// server starts accepting queries (srv.Run() below) — utils.secret is
	// a package global written once here and never again. Never logged.
	if config.DefaultConfig.DataPlane.Anonymization.Enabled {
		anonSecret := config.ResolveAnonymizationSecret(config.DefaultConfig.DataPlane.Anonymization.Secret, filepath.Dir(dbPath))
		utils.InitSecret(anonSecret)
	}

	// 2. Initialize Repositories
	repos := repositories.NewStore(db.DB)

	// 2b. Query-log retention — keep the table (and the Pi's SD card)
	// bounded. Without this the dns_queries table grows without limit.
	startQueryLogRetention(repos.QueryLogs)

	// 3. Blocklist Engine — load from DB sources, refresh periodically.
	// The DNS hot path checks an in-memory set (memBlocklist), never the
	// DB. Two independent loops keep it current:
	//   - refreshSources (below) re-fetches each enabled source's remote
	//     content on BLOCKLIST_UPDATE_INTERVAL (default 6h) — this is
	//     about the *content* of a source going stale, unrelated to local
	//     CRUD changes.
	//   - blocklistReloader.Poll, driven by startBlocklistPoll on
	//     BLOCKLIST_POLL_INTERVAL (default 5s), watches a cheap DB
	//     signature and rebuilds memBlocklist within seconds of a
	//     dashboard/CLI/MCP add, enable, disable, edit, or delete — the
	//     propagation gap policies didn't have (reloadPolicies below
	//     already polls every 5s) but blocklists did until this loop.
	blEngine := blocklist.NewEngine(repos.Blocklist)
	memBlocklist := blocklist.NewMemoryChecker()
	blReloader := newBlocklistReloader(blEngine, memBlocklist)

	// Initial load in background so DNS starts immediately. This is a
	// cheap DB-only signature check + rebuild (no network fetch), so it
	// completes fast even if refreshSources' first pass (below) is still
	// fetching remote sources.
	go blReloader.Poll()

	// Periodic re-fetch of each enabled source's remote content.
	interval, err := time.ParseDuration(config.DefaultConfig.DataPlane.BlocklistUpdateInterval)
	if err != nil || interval == 0 {
		interval = 6 * time.Hour
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		refreshSources(ctx, blEngine, blReloader)
	}()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			refreshSources(ctx, blEngine, blReloader)
			cancel()
		}
	}()

	// Fast signature poll: closes the propagation gap for local CRUD
	// changes (create/toggle/edit/delete) without waiting for the
	// refreshSources loop above. BLOCKLIST_POLL_INTERVAL, default 5s; 0
	// disables it (propagation then only happens via the initial load and
	// refreshSources passes above — the pre-existing ~6h/restart bound).
	blocklistPollInterval := envDuration("BLOCKLIST_POLL_INTERVAL", 5*time.Second)
	if blocklistPollInterval > 0 {
		startBlocklistPoll(blReloader, blocklistPollInterval)
	} else {
		logger.Log.Info("blocklist signature poll disabled (BLOCKLIST_POLL_INTERVAL=0); blocklist changes only propagate via BLOCKLIST_UPDATE_INTERVAL or restart")
	}

	// 4. Initialize Policy Engine — load from file + DB
	policyEngine := policy.NewPolicyEngine()
	policiesPath := "/app/configs/policies.json"
	if p := os.Getenv("HYDRA_POLICIES"); p != "" {
		policiesPath = p
	}
	filePolicies, err := policy.LoadPoliciesFromFile(policiesPath)
	if err != nil {
		logger.Log.Warnf("failed to load policies from file: %v (continuing with DB policies only)", err)
		filePolicies = nil
	}

	// Merge file policies + DB policies
	reloadPolicies(policyEngine, filePolicies, repos.Policies)

	// Poll DB for policy changes every 5 seconds
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			reloadPolicies(policyEngine, filePolicies, repos.Policies)
		}
	}()

	// 5. Initialize DNS Engine
	engine, err := dnsengine.NewDNSEngine(config.DefaultConfig.DataPlane, repos, policyEngine)
	if err != nil {
		logger.Log.Fatal("Failed to create DNS engine: " + err.Error())
	}

	// 6. gRPC server
	statusService := dataplanegrpc.NewStatusService(engine)
	metricsService := dataplanegrpc.NewMetricsService(engine)
	grpcSrv := dataplanegrpc.New(config.DefaultConfig.DataPlane.GRPCServer.Port, statusService, metricsService)

	go func() {
		logger.Log.Info("Starting dataplane gRPC server on :50051")
		if err := grpcSrv.Start(); err != nil {
			logger.Log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// 7. Attach blocklist checker and start DNS server
	engine.AttachBlocklistChecker(memBlocklist)
	srv, err := dnsengine.NewServer(config.DefaultConfig.DataPlane, engine)
	if err != nil {
		logger.Log.Fatal("Failed to create server: " + err.Error())
	}

	logger.Log.Infof("DNS server listening on %s", config.DefaultConfig.DataPlane.ListenAddr)
	srv.Run()
}

// refreshSources re-fetches each enabled source's remote content into the
// DB (new BlocklistSnapshot + BlocklistEntry rows on a change; a no-op on
// ETag match). It does not touch the in-memory blocklist set directly —
// the trailing reloader.Poll() call lets the single-flighted
// blocklistReloader pick up any resulting DB change (new snapshot, bumped
// source UpdatedAt) and rebuild memBlocklist, the same path the fast
// signature-poll ticker uses. Keeping exactly one rebuild path avoids two
// goroutines racing to call MemoryChecker.Reload concurrently.
func refreshSources(ctx context.Context, engine *blocklist.Engine, reloader *blocklistReloader) {
	sources, err := engine.ListSources()
	if err != nil {
		logger.Log.Errorf("Failed to list blocklist sources: %v", err)
		return
	}
	if len(sources) == 0 {
		logger.Log.Info("No blocklist sources configured")
		return
	}
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		if err := engine.UpdateSource(ctx, src, src.ETag); err != nil {
			logger.Log.Errorf("Blocklist update failed for %s: %v", src.Name, err)
		}
	}
	reloader.Poll()
}

// startQueryLogRetention runs a background loop that bounds the query log
// table by age and by row count. Both limits are configurable via env:
//
//	QUERY_LOG_RETENTION_DAYS   (default 7)   delete rows older than N days; 0 disables
//	QUERY_LOG_MAX_ROWS         (default 1e6) keep at most N newest rows; 0 disables
//	QUERY_LOG_CLEANUP_INTERVAL (default 1h)  how often to run cleanup
//
// Runs once immediately so a fat table from before this build is pruned on
// the first boot that includes retention.
func startQueryLogRetention(repo repositories.QueryLogRepository) {
	retentionDays := envInt("QUERY_LOG_RETENTION_DAYS", 7)
	maxRows := int64(envInt("QUERY_LOG_MAX_ROWS", 1_000_000))
	interval := envDuration("QUERY_LOG_CLEANUP_INTERVAL", time.Hour)

	cleanup := func() {
		if retentionDays > 0 {
			cutoff := time.Now().AddDate(0, 0, -retentionDays)
			if n, err := repo.DeleteOlderThan(cutoff); err != nil {
				logger.Log.Errorf("query log retention (age) failed: %v", err)
			} else if n > 0 {
				logger.Log.Infof("query log retention: deleted %d rows older than %d days", n, retentionDays)
			}
		}
		if maxRows > 0 {
			if n, err := repo.EnforceRowCap(maxRows); err != nil {
				logger.Log.Errorf("query log retention (cap) failed: %v", err)
			} else if n > 0 {
				logger.Log.Infof("query log retention: trimmed %d rows over cap of %d", n, maxRows)
			}
		}
	}

	go func() {
		cleanup()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			cleanup()
		}
	}()
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		logger.Log.Warnf("invalid %s=%q, using default %d", key, v, def)
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		logger.Log.Warnf("invalid %s=%q, using default %s", key, v, def)
	}
	return def
}

func reloadPolicies(engine *policy.Engine, filePolicies []policy.Policy, repo repositories.PolicyRepository) {
	// Start with file-based policies
	all := make([]policy.Policy, len(filePolicies))
	copy(all, filePolicies)

	// Add DB policies
	dbPolicies, err := repo.List()
	if err != nil {
		logger.Log.Errorf("Failed to load policies from DB: %v", err)
	} else {
		for _, dbp := range dbPolicies {
			all = append(all, dbPolicyToEngine(dbp))
		}
	}

	if err := engine.LoadPolicies(all); err != nil {
		logger.Log.Errorf("Failed to reload policy snapshot: %v", err)
	}
}

func dbPolicyToEngine(m models.Policy) policy.Policy {
	var domains []string
	if m.Domains != "" {
		_ = json.Unmarshal([]byte(m.Domains), &domains)
	}
	return policy.Policy{
		ID:       m.ID,
		Name:     m.Name,
		Action:   m.Action,
		Domains:  domains,
		Priority: m.Priority,
		Enabled:  m.Enabled,
		Category: m.Category,
		Redirect: m.RedirectIP,
	}
}
