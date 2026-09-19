package main

// SPDX-License-Identifier: GPL-3.0-or-later
import (
	"context"
	"encoding/json"
	"os"
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
)

func main() {
	logger.Log.Info("Starting PhantomDNS Data Plane...")

	// 1. Initialize DB
	dbPath := "/app/data/phantomdns.db"
	if p := os.Getenv("HYDRA_DB"); p != "" {
		dbPath = p
	}
	db.InitDB(dbPath)

	// 2. Initialize Repositories
	repos := repositories.NewStore(db.DB)

	// 2b. Query-log retention — keep the table (and the Pi's SD card)
	// bounded. Without this the dns_queries table grows without limit.
	startQueryLogRetention(repos.QueryLogs)

	// 3. Blocklist Engine — load from DB sources, refresh periodically.
	// The DNS hot path checks an in-memory set (memBlocklist), never the
	// DB; refreshBlocklists rebuilds that set after each source update.
	blEngine := blocklist.NewEngine(repos.Blocklist)
	memBlocklist := blocklist.NewMemoryChecker()

	// Initial load in background so DNS starts immediately
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		refreshBlocklists(ctx, blEngine, memBlocklist)
	}()

	// Periodic refresh
	interval, err := time.ParseDuration(config.DefaultConfig.DataPlane.BlocklistUpdateInterval)
	if err != nil || interval == 0 {
		interval = 6 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			refreshBlocklists(ctx, blEngine, memBlocklist)
			cancel()
		}
	}()

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

func refreshBlocklists(ctx context.Context, engine *blocklist.Engine, mem *blocklist.MemoryChecker) {
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
	// Rebuild the in-memory set the DNS hot path reads from.
	domains, err := engine.List()
	if err != nil {
		logger.Log.Errorf("Failed to load blocklist domains into memory: %v", err)
		return
	}
	mem.Reload(domains)
	logger.Log.Infof("Blocklist refresh complete: %d total domains blocked", mem.Count())
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
