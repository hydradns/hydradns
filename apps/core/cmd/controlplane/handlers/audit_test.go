// SPDX-License-Identifier: Apache-2.0
package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/storage/models"
)

func TestListAuditEvents_AdminSeesEverything(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)

	actor := uint(1)
	for i, act := range []string{"policy.create", "policy.delete", "blocklist.create"} {
		_ = th.store.Audit.Record(&models.AuditEvent{
			ActorID:   &actor,
			Action:    act,
			Target:    "thing",
			ClientIP:  "1.1.1.1",
			UserAgent: "ua",
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
		})
	}

	rec := th.do("GET", "/api/v1/audit", adminTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin: got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Total  int64      `json:"total"`
			Events []auditDTO `json:"events"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 3 || len(resp.Data.Events) != 3 {
		t.Errorf("expected 3 events, got total=%d events=%d", resp.Data.Total, len(resp.Data.Events))
	}
}

func TestListAuditEvents_ReadOnlyRejected(t *testing.T) {
	th := newHarness(t)
	roTok := th.seedUser(t, "ro@x.com", models.RoleReadOnly)

	rec := th.do("GET", "/api/v1/audit", roTok, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("read_only: got %d, want 403", rec.Code)
	}
}

func TestListAuditEvents_OperatorAllowed(t *testing.T) {
	th := newHarness(t)
	opTok := th.seedUser(t, "op@x.com", models.RoleOperator)

	rec := th.do("GET", "/api/v1/audit", opTok, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("operator: got %d, want 200", rec.Code)
	}
}

func TestListAuditEvents_FilterByAction(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)

	actor := uint(1)
	for _, act := range []string{"policy.create", "policy.create", "blocklist.create"} {
		_ = th.store.Audit.Record(&models.AuditEvent{
			ActorID:   &actor,
			Action:    act,
			Target:    "t",
			ClientIP:  "1.1.1.1",
			UserAgent: "ua",
		})
	}

	rec := th.do("GET", "/api/v1/audit?action=policy.create&limit=100", adminTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d", rec.Code)
	}
	var resp struct {
		Data struct {
			Total  int64      `json:"total"`
			Events []auditDTO `json:"events"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data.Total != 2 {
		t.Errorf("total: got %d, want 2", resp.Data.Total)
	}
}

func TestListAuditEvents_BadQueryParam(t *testing.T) {
	th := newHarness(t)
	adminTok := th.seedUser(t, "admin@x.com", models.RoleAdmin)

	rec := th.do("GET", "/api/v1/audit?from=not-a-date", adminTok, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad from: got %d, want 400", rec.Code)
	}
}
