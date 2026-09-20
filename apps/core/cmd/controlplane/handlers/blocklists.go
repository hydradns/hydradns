package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/internal/storage/models"
)

// Blocklist represents a blocklist source in API responses
type Blocklist struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Format       string    `json:"format"`
	Category     string    `json:"category"`
	DomainsCount int64     `json:"domains_count"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BlocklistListData struct {
	TotalBlocklists int         `json:"total_blocklists"`
	TotalDomains    int64       `json:"total_domains"`
	ActiveLists     []Blocklist `json:"active_lists"`
}

type ResponseBlocklistList struct {
	Status string            `json:"status"`
	Data   BlocklistListData `json:"data"`
	Error  *string           `json:"error"`
}

type ResponseBlocklistSingle struct {
	Status string    `json:"status"`
	Data   Blocklist `json:"data"`
	Error  *string   `json:"error"`
}

type CreateBlocklistRequest struct {
	ID       string `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	URL      string `json:"url" binding:"required"`
	Format   string `json:"format" binding:"required"`
	Category string `json:"category"`
}

// validateBlocklistURL is the one scheme check every write path that
// accepts a blocklist URL must go through — CreateBlocklist, UpdateBlocklist,
// and the setup wizard's optional blocklist bootstrap (handlers/auth.go's
// Setup). Before this it was duplicated ad hoc and only applied to some of
// them: UpdateBlocklist had the check, CreateBlocklist did not (M4 in the
// launch-prep review), meaning an operator-role user could POST a
// blocklist source pointing at an internal URL
// (http://169.254.169.254/latest/meta-data/, http://192.168.1.1/admin,
// etc.) and the appliance would fetch it.
//
// Deliberately does NOT allowlist/denylist destinations (no RFC1918,
// loopback, or link-local blocking): this is a home/office appliance where
// operator-role is a trusted role (the same trust level already extended
// to configs/policies.json on disk), not a multi-tenant service isolating
// untrusted callers from each other. Blocking obviously-wrong schemes
// (file://, ftp://, javascript:, anything that isn't http/https) closes
// the "useless or actively dangerous" cases without pretending to be a
// real SSRF allowlist. If this trust model ever changes, fetch
// destinations need real hardening (deny loopback/link-local/RFC1918/cloud
// metadata IPs) — out of scope here.
func validateBlocklistURL(raw string) error {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return fmt.Errorf("url must use http:// or https://")
	}
	return nil
}

func blocklistFromSource(src models.BlocklistSource, count int64) Blocklist {
	return Blocklist{
		ID:           src.ID,
		Name:         src.Name,
		URL:          src.URL,
		Format:       src.Format,
		Category:     src.Category,
		DomainsCount: count,
		Enabled:      src.Enabled,
		CreatedAt:    src.CreatedAt,
		UpdatedAt:    src.UpdatedAt,
	}
}

// ListBlocklists handles GET /blocklists
func (h *APIHandler) ListBlocklists(c *gin.Context) {
	sources, err := h.Store.Blocklist.ListSources()
	if err != nil {
		errMsg := "failed to fetch blocklist sources"
		c.JSON(http.StatusInternalServerError, ResponseBlocklistList{Status: "error", Error: &errMsg})
		return
	}

	counts, err := h.Store.Blocklist.CountEntriesGroupedBySource()
	if err != nil {
		counts = map[string]int64{}
	}

	var lists []Blocklist
	var totalDomains int64
	for _, src := range sources {
		count := counts[src.ID]
		totalDomains += count
		lists = append(lists, blocklistFromSource(src, count))
	}

	c.JSON(http.StatusOK, ResponseBlocklistList{
		Status: "success",
		Data: BlocklistListData{
			TotalBlocklists: len(lists),
			TotalDomains:    totalDomains,
			ActiveLists:     lists,
		},
	})
}

// GetBlocklist handles GET /blocklists/:id
func (h *APIHandler) GetBlocklist(c *gin.Context) {
	src, err := h.Store.Blocklist.GetSource(c.Param("id"))
	if err != nil {
		errMsg := "blocklist not found"
		c.JSON(http.StatusNotFound, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}
	count, _ := h.Store.Blocklist.CountEntriesBySource(src.ID)
	c.JSON(http.StatusOK, ResponseBlocklistSingle{
		Status: "success",
		Data:   blocklistFromSource(*src, count),
	})
}

// CreateBlocklist handles POST /blocklists
func (h *APIHandler) CreateBlocklist(c *gin.Context) {
	var req CreateBlocklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}

	if err := validateBlocklistURL(req.URL); err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}

	src := &models.BlocklistSource{
		ID:        req.ID,
		Name:      req.Name,
		URL:       req.URL,
		Format:    req.Format,
		Category:  req.Category,
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	if err := h.Store.Blocklist.CreateSource(src); err != nil {
		errMsg := "failed to create blocklist source"
		c.JSON(http.StatusInternalServerError, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}

	out := blocklistFromSource(*src, 0)
	h.Audit.Record(c, "blocklist.create", "blocklist:"+src.ID, nil, out)

	// Kick off the initial fetch + parse in the background so users see a
	// populated domain count within seconds. The data plane's periodic
	// refresh loop will still re-fetch on its own cadence. Duplicate fetches
	// are cheap (ETag short-circuits) and the snapshot transaction in the
	// repository guarantees consistency if both paths land simultaneously.
	if h.BlocklistEngine != nil {
		srcCopy := *src
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := h.BlocklistEngine.UpdateSource(ctx, srcCopy, ""); err != nil {
				log.Printf("initial blocklist fetch failed for %s: %v", srcCopy.ID, err)
			}
		}()
	}

	c.JSON(http.StatusCreated, ResponseBlocklistSingle{
		Status: "success",
		Data:   out,
	})
}

// UpdateBlocklistRequest carries only the fields the UI's PATCH
// /blocklists/:id can send (see apps/ui/lib/api.ts updateBlocklist and
// toggleBlocklist). Every field is a pointer so an absent field leaves
// the existing value untouched — this is a partial update, unlike
// UpdatePolicyRequest, because the only caller wired up in the shipped
// UI today (the enable/disable switch) sends a single field.
type UpdateBlocklistRequest struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	Format   *string `json:"format"`
	Category *string `json:"category"`
	Enabled  *bool   `json:"enabled"`
}

// UpdateBlocklist handles PATCH /blocklists/:id.
//
// Propagation: exactly the same mechanism CreateBlocklist already uses —
// on a URL or format change, an immediate background fetch is kicked off
// via BlocklistEngine.UpdateSource, which refreshes the DB (BlocklistEntry
// rows). Separately, the dataplane's in-memory blocklist set (what the DNS
// hot path actually checks) is kept in sync by a lightweight signature
// poll in cmd/dataplane (BlocklistSignature, default BLOCKLIST_POLL_INTERVAL=
// 5s): it detects a source/entry change cheaply (without scanning
// blocklist_entries) and rebuilds the in-memory set from enabled sources
// only when the signature changes, so an edit here — including flipping
// Enabled — reaches the DNS engine within about 5 seconds, not the full
// BLOCKLIST_UPDATE_INTERVAL (default 6h) periodic refresh window.
//
// Ingested-entries semantics on a URL/format change: BlocklistEntry rows
// are never deleted or replaced in place when a source's content changes
// — SaveSnapshotWithEntries (used both by this refetch and by the normal
// periodic refresh) only ever appends a new BlocklistSnapshot + its
// entries for the source ID. So after editing a source's URL, entries
// from the OLD url remain in the table alongside the new ones until the
// source itself is deleted (which does cascade-delete all of its
// snapshots/entries). This is not a new limitation introduced here: it is
// the existing refresh semantics, identical to what happens today every
// time the periodic 6h refresh re-fetches changed content at an
// unchanged URL.
func (h *APIHandler) UpdateBlocklist(c *gin.Context) {
	id := c.Param("id")

	src, err := h.Store.Blocklist.GetSource(id)
	if err != nil || src == nil {
		errMsg := "blocklist not found"
		c.JSON(http.StatusNotFound, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}
	beforeCount, _ := h.Store.Blocklist.CountEntriesBySource(src.ID)
	before := blocklistFromSource(*src, beforeCount)

	var req UpdateBlocklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}

	refetch := false
	if req.Name != nil {
		src.Name = *req.Name
	}
	if req.URL != nil && *req.URL != src.URL {
		if err := validateBlocklistURL(*req.URL); err != nil {
			errMsg := err.Error()
			c.JSON(http.StatusBadRequest, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
			return
		}
		src.URL = *req.URL
		src.ETag = "" // the old ETag belongs to the old URL; force a fresh fetch
		refetch = true
	}
	if req.Format != nil && *req.Format != src.Format {
		src.Format = *req.Format
		refetch = true
	}
	if req.Category != nil {
		src.Category = *req.Category
	}
	if req.Enabled != nil {
		src.Enabled = *req.Enabled
	}
	src.UpdatedAt = time.Now()

	if err := h.Store.Blocklist.UpdateSourceFields(src); err != nil {
		errMsg := "failed to update blocklist source"
		c.JSON(http.StatusInternalServerError, ResponseBlocklistSingle{Status: "error", Error: &errMsg})
		return
	}

	afterCount, _ := h.Store.Blocklist.CountEntriesBySource(src.ID)
	out := blocklistFromSource(*src, afterCount)
	h.Audit.Record(c, "blocklist.update", "blocklist:"+id, before, out)

	// Same async-fetch pattern as CreateBlocklist: don't make the operator
	// wait for the next periodic refresh to see a URL/format edit reflected
	// in domains_count.
	if refetch && h.BlocklistEngine != nil {
		srcCopy := *src
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := h.BlocklistEngine.UpdateSource(ctx, srcCopy, ""); err != nil {
				log.Printf("blocklist re-fetch after edit failed for %s: %v", srcCopy.ID, err)
			}
		}()
	}

	c.JSON(http.StatusOK, ResponseBlocklistSingle{
		Status: "success",
		Data:   out,
	})
}

// DeleteBlocklist handles DELETE /blocklists/:id
func (h *APIHandler) DeleteBlocklist(c *gin.Context) {
	id := c.Param("id")
	var before *Blocklist
	if src, err := h.Store.Blocklist.GetSource(id); err == nil && src != nil {
		b := blocklistFromSource(*src, 0)
		before = &b
	}

	if err := h.Store.Blocklist.DeleteSource(id); err != nil {
		errMsg := "blocklist not found"
		c.JSON(http.StatusNotFound, ResponseGeneric{Status: "error", Error: &errMsg})
		return
	}

	h.Audit.Record(c, "blocklist.delete", "blocklist:"+id, before, nil)

	c.JSON(http.StatusOK, ResponseGeneric{Status: "success", Data: map[string]interface{}{}})
}
