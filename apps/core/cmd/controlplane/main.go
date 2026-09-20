// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/hydradns/hydra-core/cmd/controlplane/demoseed"
	"github.com/hydradns/hydra-core/cmd/controlplane/handlers"
	"github.com/hydradns/hydra-core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydra-core/cmd/controlplane/routes"
	"github.com/hydradns/hydra-core/internal/blocklist"
	"github.com/hydradns/hydra-core/internal/config"
	client "github.com/hydradns/hydra-core/internal/grpc/controlplane"
	"github.com/hydradns/hydra-core/internal/storage/db"
	"github.com/hydradns/hydra-core/internal/storage/repositories"

	"github.com/gin-gonic/gin"
)

// demoRefreshInterval controls how often a running public demo re-anchors
// its synthetic query-log timestamps to "now" (see demoseed.Refresh) so a
// long-lived container doesn't drift into showing stale charts.
const demoRefreshInterval = 30 * time.Minute

func main() {
	// Initialize database
	dbPath := db.ResolveDBPath(os.Getenv("HYDRA_DB"))
	db.InitDB(dbPath)
	repos := repositories.NewStore(db.DB)

	// HYDRA_DEMO_MODE=true turns this instance into a public, read-only
	// demo: see middlewares.DemoGuard (the actual write-blocking boundary)
	// and cmd/controlplane/demoseed (the seeded user + synthetic data).
	// Default is off, and when off none of this file's demo-mode branches
	// run — behaviour is byte-for-byte the same as before this feature
	// existed.
	demoMode := strings.EqualFold(os.Getenv("HYDRA_DEMO_MODE"), "true")
	if demoMode {
		// Refuses to proceed (fatal) if this looks like a real deployment's
		// database rather than a fresh demo volume — see the doc comment
		// on EnsureDemoUser for why that matters.
		if err := demoseed.EnsureDemoUser(repos); err != nil {
			log.Fatalf("demo mode startup check failed: %v", err)
		}
		if err := demoseed.SeedIfEmpty(repos); err != nil {
			log.Fatalf("demo data seed failed: %v", err)
		}
		stop := make(chan struct{})
		defer close(stop)
		go demoseed.StartRefreshLoop(repos, db.DB, demoRefreshInterval, stop)
	}

	// Initialize grpc client
	c, err := client.New(config.DefaultConfig.DataPlane.GRPCServer.ListenAddr)
	if err != nil {
		log.Fatalf("failed to connect to dataplane: %v", err)
	}
	defer c.Close()

	// Load the persistent configuration
	state, err := repos.SystemState.Get()
	if err != nil {
		log.Fatalf("failed to load system state: %v", err)
	}
	c.SetAcceptQueries(state.DNSEnabled)

	// Blocklist engine powers the immediate fetch on POST /blocklists so
	// users don't wait up to 6 hours for the next dataplane refresh cycle.
	blocklistEngine := blocklist.NewEngine(repos.Blocklist)

	// Initialize Gin router
	apiHandler := handlers.NewAPIHandler(*repos, c, blocklistEngine, demoMode)
	r := gin.Default()

	// gin trusts every proxy by default, which means c.ClientIP() (used
	// both by the audit log and by the login/setup rate limiter) would
	// honor a client-supplied X-Forwarded-For header, letting any caller
	// spoof its own client IP: dodge the per-IP login throttle, or make
	// the audit log record a fabricated address for its own actions.
	// TRUSTED_PROXIES (comma-separated CIDRs/IPs) is empty by default,
	// meaning "trust none" — c.ClientIP() then always resolves to the
	// real socket address. Only set it if this API sits behind a reverse
	// proxy that itself overwrites (never appends to) X-Forwarded-For
	// before forwarding, and set it to that proxy's address specifically
	// (the demo's own reverse proxy included — see demo/README.md).
	if err := configureTrustedProxies(r); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}

	buildMiddlewareChain(r, demoMode, repos.Users, repos.Tokens)

	routes.RegisterRoutes(r, apiHandler, demoMode)
	r.Run(config.DefaultConfig.ControlPlane.ListenAddr)
}

// buildMiddlewareChain installs the global middleware stack in the
// production order: Logger -> CORS -> [DemoGuard] -> Auth. DemoGuard is
// only appended when demoMode is true, so a disabled demo mode is not
// merely a no-op guard sitting in the chain — the middleware is not
// installed at all, and TestBuildMiddlewareChain_DemoGuardOnlyWhenEnabled
// asserts exactly that.
//
// DemoGuard must run before Auth so it cannot be bypassed by any role,
// including admin — see the doc comment on middlewares.DemoGuard.
func buildMiddlewareChain(r *gin.Engine, demoMode bool, users repositories.UserRepository, tokens repositories.TokenRepository) {
	r.Use(middlewares.Logger())

	// CORS middleware (development-friendly). See cmd/controlplane/middlewares/cors.go
	r.Use(middlewares.CORS())

	if demoMode {
		r.Use(middlewares.DemoGuard())
	}

	// Auth middleware — validates Bearer token on protected routes
	r.Use(middlewares.Auth(users, tokens))
}

// configureTrustedProxies applies TRUSTED_PROXIES (comma-separated
// CIDRs/IPs) to r, defaulting to nil (trust no proxies) when unset. nil
// is a distinct value from an empty-but-non-nil slice for gin's
// SetTrustedProxies — passing nil is what actually disables proxy
// trust; an empty slice is not equivalent.
func configureTrustedProxies(r *gin.Engine) error {
	var proxies []string
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				proxies = append(proxies, p)
			}
		}
	}
	return r.SetTrustedProxies(proxies)
}
