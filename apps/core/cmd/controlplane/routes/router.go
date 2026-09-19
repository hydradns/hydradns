package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/cmd/controlplane/handlers"
	"github.com/hydradns/hydra-core/cmd/controlplane/middlewares"
	"github.com/hydradns/hydra-core/internal/storage/models"
)

// writeRoles names the roles permitted to mutate policies, blocklists,
// and the engine toggle. Admins bypass role checks entirely via the
// middleware, so "operator" here is effectively "operator or admin".
var writeRoles = []string{models.RoleOperator}

// RegisterRoutes registers every control plane endpoint. Read routes are
// open to any authenticated user (read_only included). Mutating routes
// wrap the handler in middlewares.RequireRole so the role boundary lives
// next to the route definition, not inside the handler.
func RegisterRoutes(r *gin.Engine, apiHandler *handlers.APIHandler) {
	api := r.Group("/api/v1")
	r.GET("/health", apiHandler.HealthCheck)
	r.GET("/", apiHandler.Root)
	{
		// Auth endpoints (unprotected — middleware exempts these paths)
		auth := api.Group("/auth")
		{
			auth.GET("/status", apiHandler.GetAuthStatus)
			auth.POST("/setup", apiHandler.Setup)
			auth.POST("/login", apiHandler.Login)
		}

		// Dashboard endpoints (read-only, open to all authenticated users)
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/summary", apiHandler.GetDashboardSummary)
		}

		// DNS Engine endpoints — read open, write requires operator+
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
			blocklists.DELETE("/:id",
				middlewares.RequireRole(writeRoles...),
				apiHandler.DeleteBlocklist)
		}

		// Analytics endpoints
		analytics := api.Group("/analytics")
		{
			analytics.GET("/summary", apiHandler.GetAnalyticsSummary)
			analytics.GET("/audits", apiHandler.GetAuditLogs)
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
