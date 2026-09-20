// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildDemoGuardRouter mounts DemoGuard in front of the same routes the
// production router registers for /auth, plus a representative mutating
// route (POST /api/v1/policies) — enough to prove the guard blocks
// mutations generically rather than by an allowlist of blocked paths.
func buildDemoGuardRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DemoGuard())
	r.GET("/api/v1/policies", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/policies", func(c *gin.Context) { c.Status(http.StatusCreated) })
	r.PUT("/api/v1/policies/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.DELETE("/api/v1/policies/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/auth/setup", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusCreated) })
	return r
}

func doDemoReq(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestDemoGuard_BlocksMutatingMethods(t *testing.T) {
	r := buildDemoGuardRouter()
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/api/v1/policies"},
		{http.MethodPut, "/api/v1/policies/block-ads"},
		{http.MethodDelete, "/api/v1/policies/block-ads"},
		{http.MethodPost, "/api/v1/users"},
	}
	for _, tc := range cases {
		rec := doDemoReq(r, tc.method, tc.path)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: got %d, want 403", tc.method, tc.path, rec.Code)
		}
		if body := rec.Body.String(); !strings.Contains(body, "demo mode: changes are disabled") {
			t.Errorf("%s %s: unexpected body %q", tc.method, tc.path, body)
		}
	}
}

func TestDemoGuard_AllowsSafeMethods(t *testing.T) {
	r := buildDemoGuardRouter()
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		rec := doDemoReq(r, method, "/api/v1/policies")
		if rec.Code == http.StatusForbidden {
			t.Errorf("%s /api/v1/policies: got 403, want it to pass through", method)
		}
	}
}

func TestDemoGuard_AllowsLogin(t *testing.T) {
	r := buildDemoGuardRouter()
	rec := doDemoReq(r, http.MethodPost, "/api/v1/auth/login")
	if rec.Code != http.StatusOK {
		t.Errorf("POST /api/v1/auth/login: got %d, want 200 (allowlisted)", rec.Code)
	}
}

func TestDemoGuard_BlocksSetup(t *testing.T) {
	r := buildDemoGuardRouter()
	rec := doDemoReq(r, http.MethodPost, "/api/v1/auth/setup")
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /api/v1/auth/setup: got %d, want 403 (never allowlisted)", rec.Code)
	}
}
