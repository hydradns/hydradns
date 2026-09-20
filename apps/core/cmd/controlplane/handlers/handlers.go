package handlers

import (
	"github.com/hydradns/hydra-core/cmd/controlplane/audit"
	"github.com/hydradns/hydra-core/internal/blocklist"
	client "github.com/hydradns/hydra-core/internal/grpc/controlplane"
	"github.com/hydradns/hydra-core/internal/storage/repositories"
)

// APIHandler contains dependencies for API endpoints
type APIHandler struct {
	Store           repositories.Store
	DataPlaneClient *client.Client
	// BlocklistEngine is used to kick off an immediate fetch when a new
	// source is created via the API, so users see domain counts within
	// seconds instead of waiting for the data plane's periodic refresh.
	BlocklistEngine *blocklist.Engine
	// Audit records one event per mutating action. Lives on the handler
	// so every mutation has a single call site (h.Audit.Record(...)).
	Audit *audit.Recorder
	// DemoMode mirrors HYDRA_DEMO_MODE. It never gates write access on its
	// own (that boundary is middlewares.DemoGuard, installed ahead of
	// Auth) — it only controls two read-side, non-security-critical
	// behaviors: GET /api/v1/auth/status advertising demo_mode to the UI,
	// and client-IP redaction on the query-log / audit / bypass responses
	// (see maskClientIP in common.go). Defaults to false so every existing
	// caller of NewAPIHandler, and every test constructing &APIHandler{}
	// directly, is unaffected.
	DemoMode bool
	// AnonymizeSecret is the HMAC key used to hash a client-IP filter
	// value in GET /analytics/logs before querying, so the filter still
	// works when HYDRA_ANONYMIZE_CLIENT_IPS is on (see
	// resolveClientIPFilter in common.go). Empty means anonymization is
	// disabled: the raw filter value is used as-is, unchanged from
	// before this field existed. Resolved once in main.go via
	// config.ResolveAnonymizationSecret — the same secret (and the same
	// HMAC-SHA256-first-16-hex-chars algorithm as utils.AnonymizeIP) the
	// dataplane used to write client_ip, so a hash computed here matches
	// a hash already stored in the table.
	AnonymizeSecret string
}

func NewAPIHandler(
	store repositories.Store,
	dataPlaneClient *client.Client,
	blocklistEngine *blocklist.Engine,
	demoMode bool,
	anonymizeSecret string,
) *APIHandler {
	return &APIHandler{
		Store:           store,
		DataPlaneClient: dataPlaneClient,
		BlocklistEngine: blocklistEngine,
		Audit:           audit.New(store.Audit),
		DemoMode:        demoMode,
		AnonymizeSecret: anonymizeSecret,
	}
}
