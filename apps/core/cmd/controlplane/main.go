// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"log"
	"os"
	"strings"

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

func main() {
	// Initialize database
	dbPath := db.ResolveDBPath(os.Getenv("HYDRA_DB"))
	db.InitDB(dbPath)
	repos := repositories.NewStore(db.DB)

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
	apiHandler := handlers.NewAPIHandler(*repos, c, blocklistEngine)
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
	// before forwarding, and set it to that proxy's address specifically.
	if err := configureTrustedProxies(r); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}

	r.Use(middlewares.Logger())

	// CORS middleware (development-friendly). See cmd/controlplane/middlewares/cors.go
	r.Use(middlewares.CORS())

	// Auth middleware — validates Bearer token on protected routes
	r.Use(middlewares.Auth(repos.Users, repos.Tokens))

	routes.RegisterRoutes(r, apiHandler)
	r.Run(config.DefaultConfig.ControlPlane.ListenAddr)
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
