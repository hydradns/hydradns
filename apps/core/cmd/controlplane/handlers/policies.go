package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydra-core/internal/storage/models"
	"gorm.io/gorm"
)

// Policy represents a policy in API responses
type Policy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Action      string   `json:"action"`
	RedirectIP  string   `json:"redirect_ip,omitempty"`
	Domains     []string `json:"domains"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
}

type PolicyListData struct {
	TotalPolicies    int      `json:"total_policies"`
	ActivePolicies   int      `json:"active_policies"`
	InactivePolicies int      `json:"inactive_policies"`
	List             []Policy `json:"list"`
}

type ResponsePolicyList struct {
	Status string         `json:"status"`
	Data   PolicyListData `json:"data"`
	Error  *string        `json:"error"`
}

type ResponsePolicySingle struct {
	Status string  `json:"status"`
	Data   Policy  `json:"data"`
	Error  *string `json:"error"`
}

type CreatePolicyRequest struct {
	ID          string   `json:"id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Action      string   `json:"action" binding:"required"`
	RedirectIP  string   `json:"redirect_ip"`
	Domains     []string `json:"domains" binding:"required"`
	Priority    int      `json:"priority"`
}

// validatePolicyAction checks action (and, for REDIRECT, redirectIP)
// against what the dataplane's policy engine and file loader actually
// understand — BLOCK/ALLOW/REDIRECT, matched case-insensitively (see
// internal/policy/engine.go's policyDecision switch and
// internal/policy/loader.go's ValidatePolicy, which apply the identical
// rule to configs/policies.json). The engine's switch silently falls
// through to ActionAllow for anything it doesn't recognize, so without
// this check, POST/PUT with a typo'd action (e.g. "DENY") returns 200 and
// a policy that was supposed to BLOCK a category starts silently allowing
// it (M10 in the launch-prep review). REDIRECT additionally requires a
// real IP in redirect_ip — the field the dataplane forwards matched
// queries to — since an empty or unparseable target is not a usable
// redirect either.
func validatePolicyAction(action, redirectIP string) (errMsg string, ok bool) {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "BLOCK", "ALLOW":
		return "", true
	case "REDIRECT":
		redirectIP = strings.TrimSpace(redirectIP)
		if redirectIP == "" {
			return "redirect action requires a non-empty redirect_ip", false
		}
		if net.ParseIP(redirectIP) == nil {
			return fmt.Sprintf("invalid redirect_ip %q: must be a valid IP address", redirectIP), false
		}
		return "", true
	default:
		return fmt.Sprintf("unsupported action %q: must be one of BLOCK, ALLOW, REDIRECT", action), false
	}
}

func policyFromModel(m models.Policy) Policy {
	var domains []string
	if m.Domains != "" {
		_ = json.Unmarshal([]byte(m.Domains), &domains)
	}
	return Policy{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Category:    m.Category,
		Action:      m.Action,
		RedirectIP:  m.RedirectIP,
		Domains:     domains,
		Priority:    m.Priority,
		Enabled:     m.Enabled,
	}
}

// ListPolicies handles GET /policies
func (h *APIHandler) ListPolicies(c *gin.Context) {
	models, err := h.Store.Policies.List()
	if err != nil {
		errMsg := "failed to fetch policies"
		c.JSON(http.StatusInternalServerError, ResponsePolicyList{Status: "error", Error: &errMsg})
		return
	}

	var list []Policy
	activeCount := 0
	for _, m := range models {
		p := policyFromModel(m)
		list = append(list, p)
		if p.Enabled {
			activeCount++
		}
	}

	c.JSON(http.StatusOK, ResponsePolicyList{
		Status: "success",
		Data: PolicyListData{
			TotalPolicies:    len(list),
			ActivePolicies:   activeCount,
			InactivePolicies: len(list) - activeCount,
			List:             list,
		},
	})
}

// GetPolicy handles GET /policies/:id
func (h *APIHandler) GetPolicy(c *gin.Context) {
	m, err := h.Store.Policies.GetByID(c.Param("id"))
	if err != nil {
		errMsg := "policy not found"
		c.JSON(http.StatusNotFound, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}
	c.JSON(http.StatusOK, ResponsePolicySingle{
		Status: "success",
		Data:   policyFromModel(*m),
	})
}

// CreatePolicy handles POST /policies
func (h *APIHandler) CreatePolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	if errMsg, ok := validatePolicyAction(req.Action, req.RedirectIP); !ok {
		c.JSON(http.StatusBadRequest, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	domainsJSON, _ := json.Marshal(req.Domains)

	m := &models.Policy{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Action:      req.Action,
		RedirectIP:  req.RedirectIP,
		Domains:     string(domainsJSON),
		Priority:    req.Priority,
		Enabled:     true,
	}
	if err := h.Store.Policies.Create(m); err != nil {
		errMsg := "failed to create policy"
		c.JSON(http.StatusInternalServerError, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	out := policyFromModel(*m)
	h.Audit.Record(c, "policy.create", "policy:"+m.ID, nil, out)

	c.JSON(http.StatusCreated, ResponsePolicySingle{
		Status: "success",
		Data:   out,
	})
}

// UpdatePolicyRequest is deliberately distinct from CreatePolicyRequest:
// it has no ID field at all (the path :id always wins — there is nowhere
// for a body id to even bind to), but otherwise requires the same fields
// create does, since the UI's Edit Policy drawer always sends the full
// resource, not a partial patch (see apps/ui/app/dashboard/policies/page.tsx
// handleSubmit — the same `payload` object is sent for both create and
// edit; only the presence of `id` differs).
type UpdatePolicyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Action      string   `json:"action" binding:"required"`
	RedirectIP  string   `json:"redirect_ip"`
	Domains     []string `json:"domains" binding:"required"`
	Priority    int      `json:"priority"`
	// Enabled is a pointer so an omitted field preserves the existing
	// row's value instead of resetting it to false.
	Enabled *bool `json:"enabled"`
}

// UpdatePolicy handles PUT /policies/:id (the Edit Policy drawer).
//
// Propagation: the dataplane polls PolicyRepository.List() into its
// in-memory PolicySnapshot every 5s (see cmd/dataplane/main.go's
// reloadPolicies ticker) — the exact same mechanism CreatePolicy and
// DeletePolicy already rely on. An edit here needs no new propagation
// path: the next poll (within 5s) picks up the row this handler wrote.
func (h *APIHandler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.Store.Policies.GetByID(id)
	if err != nil || existing == nil {
		errMsg := "policy not found"
		c.JSON(http.StatusNotFound, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}
	before := policyFromModel(*existing)

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadRequest, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	if errMsg, ok := validatePolicyAction(req.Action, req.RedirectIP); !ok {
		c.JSON(http.StatusBadRequest, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	domainsJSON, _ := json.Marshal(req.Domains)

	updated := *existing // preserves CreatedAt and any fields this DTO doesn't carry
	updated.ID = id      // path wins, unconditionally
	updated.Name = req.Name
	updated.Description = req.Description
	updated.Category = req.Category
	updated.Action = req.Action
	updated.RedirectIP = req.RedirectIP
	updated.Domains = string(domainsJSON)
	updated.Priority = req.Priority
	updated.Enabled = enabled

	if err := h.Store.Policies.Update(&updated); err != nil {
		errMsg := "failed to update policy"
		c.JSON(http.StatusInternalServerError, ResponsePolicySingle{Status: "error", Error: &errMsg})
		return
	}

	out := policyFromModel(updated)
	h.Audit.Record(c, "policy.update", "policy:"+id, before, out)

	c.JSON(http.StatusOK, ResponsePolicySingle{
		Status: "success",
		Data:   out,
	})
}

// DeletePolicy handles DELETE /policies/:id
func (h *APIHandler) DeletePolicy(c *gin.Context) {
	// Capture the pre-delete row for the audit trail. A miss here is
	// fine; we surface a clean 404 below.
	id := c.Param("id")
	var before *Policy
	if m, err := h.Store.Policies.GetByID(id); err == nil && m != nil {
		p := policyFromModel(*m)
		before = &p
	}

	if err := h.Store.Policies.Delete(id); err != nil {
		status := http.StatusInternalServerError
		errMsg := "failed to delete policy"
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
			errMsg = "policy not found"
		}
		c.JSON(status, ResponseGeneric{Status: "error", Error: &errMsg})
		return
	}

	h.Audit.Record(c, "policy.delete", "policy:"+id, before, nil)

	c.JSON(http.StatusOK, ResponseGeneric{Status: "success", Data: map[string]interface{}{}})
}
