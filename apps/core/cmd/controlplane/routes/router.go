package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/cmd/controlplane/handlers"
	"github.com/hydradns/hydra-core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydra-core/internal/storage/models"
)

// loginRateLimit bounds POST /auth/login and /auth/setup: 10 attempts per
// 5 minutes per client IP, shared across both endpoints (an attacker
// hammering /setup counts against the same budget as one hammering
// /login). maxTrackedIPs bounds the limiter's memory regardless of how
// many distinct source IPs are seen.
const (
	loginRateLimitAttempts = 10
	loginRateLimitWindow   = 5 * time.Minute
	loginRateLimitMaxIPs   = 10000

	// demoLoginRateLimitAttempts relaxes the login budget when
	// HYDRA_DEMO_MODE=true. The demo password is fixed and publicly
	// documented (see demo/README.md), so throttling it tighter buys no
	// confidentiality; the only remaining purpose of a limit here is
	// abuse/log-spam protection, not secrecy. A public demo is commonly
	// reached by many visitors behind one shared NAT/proxy IP (an office,
	// a campus, or the demo's own reverse proxy if TRUSTED_PROXIES isn't
	// configured for it; see demo/README.md), and 10 attempts/5min shared
	// across all of them would routinely lock everyone out over one
	// visitor's "Enter Demo" clicks. A higher ceiling keeps a bound in
	// place while tolerating that.
	demoLoginRateLimitAttempts = 100
)

// writeRoles names the roles permitted to mutate policies, blocklists,
// and the engine toggle. Admins bypass role checks entirely via the
// middleware, so "operator" here is effectively "operator or admin".
var writeRoles = []string{models.RoleOperator}

// RegisterRoutes registers every control plane endpoint. Read routes are
// open to any authenticated user (read_only included). Mutating routes
// wrap the handler in middlewares.RequireRole so the role boundary lives
// next to the route definition, not inside the handler.
//
// demoMode relaxes the login rate limit (see demoLoginRateLimitAttempts);
// it does not change anything else about the route table. The actual
// write-blocking boundary for a demo deployment is middlewares.DemoGuard,
// installed in main.go ahead of this function being called.
func RegisterRoutes(r *gin.Engine, apiHandler *handlers.APIHandler, demoMode bool) {
	api := r.Group("/api/v1")
	r.GET("/health", apiHandler.HealthCheck)
	r.GET("/", apiHandler.Root)
	{
		// Auth endpoints (unprotected: middleware exempts these paths).
		// /setup and /login are additionally throttled per client IP:
		// they are the only endpoints an unauthenticated caller can hit
		// repeatedly to brute-force a password. See middlewares.LoginThrottle
		// for why this depends on main.go's r.SetTrustedProxies(nil).
		loginAttempts := loginRateLimitAttempts
		if demoMode {
			loginAttempts = demoLoginRateLimitAttempts
		}
		loginLimiter := middlewares.NewRateLimiter(loginAttempts, loginRateLimitWindow, loginRateLimitMaxIPs)
		auth := api.Group("/auth")
		{
			auth.GET("/status", apiHandler.GetAuthStatus)
			auth.POST("/setup", middlewares.LoginThrottle(loginLimiter), apiHandler.Setup)
			auth.POST("/login", middlewares.LoginThrottle(loginLimiter), apiHandler.Login)
		}

		// Dashboard endpoints (read-only, open to all authenticated users)
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/summary", apiHandler.GetDashboardSummary)
		}

		// DNS Engine endpoints: read open, write requires operator+
		dns := api.Group("/dns")
		{
			dns.GET("/engine", apiHandler.GetDnsEngineStatus)
			dns.POST("/engine",
				middlewares.RequireRole(writeRoles...),
				apiHandler.ToggleDnsEngine)
			dns.GET("/resolvers", apiHandler.ListResolvers)
			dns.GET("/metrics", apiHandler.GetDnsMetrics)
		}

		// Policies endpoints
		policies := api.Group("/policies")
		{
			policies.GET("", apiHandler.ListPolicies)
			policies.POST("",
				middlewares.RequireRole(writeRoles...),
				apiHandler.CreatePolicy)
			policies.GET("/:id", apiHandler.GetPolicy)
			policies.PUT("/:id",
				middlewares.RequireRole(writeRoles...),
				apiHandler.UpdatePolicy)
			policies.DELETE("/:id",
				middlewares.RequireRole(writeRoles...),
				apiHandler.DeletePolicy)
		}

		// Blocklists endpoints
		blocklists := api.Group("/blocklists")
		{
			blocklists.GET("", apiHandler.ListBlocklists)
			blocklists.POST("",
				middlewares.RequireRole(writeRoles...),
				apiHandler.CreateBlocklist)
			blocklists.GET("/:id", apiHandler.GetBlocklist)
			blocklists.PATCH("/:id",
				middlewares.RequireRole(writeRoles...),
				apiHandler.UpdateBlocklist)
			blocklists.DELETE("/:id",
				middlewares.RequireRole(writeRoles...),
				apiHandler.DeleteBlocklist)
		}

		// Analytics endpoints
		analytics := api.Group("/analytics")
		{
			analytics.GET("/summary", apiHandler.GetAnalyticsSummary)
			analytics.GET("/audits", apiHandler.GetAuditLogs)
			analytics.GET("/logs", apiHandler.GetQueryLogsPage)
			analytics.GET("/bypass", apiHandler.GetBypassAttempts)
		}

		// Audit log: who did what to the control plane.
		// Admin + operator can read; read_only cannot (it's sensitive).
		api.GET("/audit",
			middlewares.RequireRole(models.RoleOperator),
			apiHandler.ListAuditEvents)

		// User management endpoints. Read endpoints are open to any
		// authenticated caller; writes are admin-only except PATCH
		// which the handler itself gates (admin for any user, self for
		// own email / password).
		users := api.Group("/users")
		{
			users.GET("/me", apiHandler.GetMe)
			users.GET("", apiHandler.ListUsers)
			users.POST("",
				middlewares.RequireRole(models.RoleAdmin),
				apiHandler.CreateUser)
			users.PATCH("/:id", apiHandler.PatchUser)
			users.POST("/:id/disable",
				middlewares.RequireRole(models.RoleAdmin),
				apiHandler.SetUserDisabled)
			users.DELETE("/:id",
				middlewares.RequireRole(models.RoleAdmin),
				apiHandler.DeleteUser)
		}

		// Token management endpoints. All authenticated users can
		// manage their own tokens; admins can list+revoke any.
		tokens := api.Group("/tokens")
		{
			tokens.GET("", apiHandler.ListTokens)
			tokens.POST("", apiHandler.CreateToken)
			tokens.DELETE("/:id", apiHandler.RevokeToken)
		}
	}
}
