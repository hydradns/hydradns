package handlers

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// maskClientIP redacts the low-order bits of a client IP for demo-mode
// responses, e.g. "192.168.1.42" -> "192.168.1.x", "192.168.1.42:5353" ->
// "192.168.1.x:5353". Used by every response that carries a real client_ip
// field (query logs, the DNS-bypass panel, the RBAC audit log) when
// APIHandler.DemoMode is true.
//
// Belt-and-suspenders: in a correctly-run public demo (HYDRA_DEMO_MODE=true,
// DNS ports unpublished per demo/docker-compose.demo.yml) every row in
// these tables is synthetic seed data to begin with — see
// cmd/controlplane/demoseed — because no real DNS traffic is ever
// forwarded and mutating requests (the only way to add new rows outside
// the DNS hot path) are rejected by middlewares.DemoGuard before they
// reach a handler. This redaction does not depend on that invariant
// holding, though: it applies uniformly to whatever is actually in the
// table, so a misconfigured demo deployment (e.g. someone forgets to
// unpublish port 53, or a future change starts auditing reads) still does
// not leak a real visitor's IP address through these endpoints.
func maskClientIP(ip string) string {
	if ip == "" {
		return ip
	}

	host, port, err := net.SplitHostPort(ip)
	if err != nil {
		host, port = ip, ""
	}

	parsed := net.ParseIP(host)
	if parsed == nil {
		// Not a parseable IP (e.g. already redacted, or malformed legacy
		// data) — do not echo it back unmasked.
		return "x.x.x.x"
	}

	var masked string
	if v4 := parsed.To4(); v4 != nil {
		masked = fmt.Sprintf("%d.%d.%d.x", v4[0], v4[1], v4[2])
	} else {
		segs := strings.Split(parsed.String(), ":")
		segs[len(segs)-1] = "x"
		masked = strings.Join(segs, ":")
	}

	if port != "" {
		return masked + ":" + port
	}
	return masked
}

func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}

func (h *APIHandler) Root(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Welcome to HydraDNS Control Plane API",
	})
}
