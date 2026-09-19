package handlers

import (
	"github.com/lopster568/phantomDNS/cmd/controlplane/audit"
	"github.com/lopster568/phantomDNS/internal/blocklist"
	client "github.com/lopster568/phantomDNS/internal/grpc/controlplane"
	"github.com/lopster568/phantomDNS/internal/storage/repositories"
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
}

func NewAPIHandler(
	store repositories.Store,
	dataPlaneClient *client.Client,
	blocklistEngine *blocklist.Engine,
) *APIHandler {
	return &APIHandler{
		Store:           store,
		DataPlaneClient: dataPlaneClient,
		BlocklistEngine: blocklistEngine,
		Audit:           audit.New(store.Audit),
	}
}
