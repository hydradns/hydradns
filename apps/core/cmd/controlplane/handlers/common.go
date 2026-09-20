package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// resolveClientIPFilter turns the raw `client=` query value from GET
// /analytics/logs into what should actually be compared against the
// stored client_ip column, given demo mode and anonymization (M3 and M8
// in the launch-prep review). raw is assumed already non-empty and
// trimmed by the caller.
//
// Demo mode: responses mask client_ip to "x.x.x.x" / "192.168.1.x" (see
// maskClientIP above), but the underlying rows are unmasked, and an
// exact-match filter compares against the real value. That turns the
// filter into a 256-guess oracle for the last octet a masked IP is
// supposedly hiding — so demo mode rejects the filter outright rather
// than silently ignoring it (silently ignoring it would return unfiltered
// results under a URL that looks filtered, which is its own kind of
// wrong).
//
// Anonymization: when HYDRA_ANONYMIZE_CLIENT_IPS is on, the dataplane
// stores an HMAC-SHA256 of the client IP (first 16 hex chars — see
// utils.AnonymizeIP), not the raw address, so `client_ip = <raw ip>` can
// never match a row. h.AnonymizeSecret (resolved in main.go via
// config.ResolveAnonymizationSecret, the same secret file the dataplane
// reads) lets this hash the filter value the same way before querying.
func (h *APIHandler) resolveClientIPFilter(raw string) (string, error) {
	if h.DemoMode {
		return "", fmt.Errorf("client filtering is unavailable in demo mode (client IPs are masked in responses)")
	}
	if h.AnonymizeSecret == "" {
		return raw, nil
	}
	hashed, ok := hashClientIPForFilter(h.AnonymizeSecret, raw)
	if !ok {
		return "", fmt.Errorf("invalid client filter value %q", raw)
	}
	return hashed, nil
}

// hashClientIPForFilter mirrors utils.AnonymizeIP's hashed path exactly
// (HMAC-SHA256 over the 16-byte form of the address, first 16 hex chars)
// without depending on that package's process-global secret variable —
// this runs in the control plane, a separate process from the dataplane
// that actually wrote the hashes being compared against, so there is no
// shared state to piggyback on (and no reason to introduce any: the
// secret is passed in explicitly, resolved once in main.go). ok is false
// when raw does not parse as an IP (with or without a port suffix), so
// callers can 400 instead of silently comparing against a filter that can
// never match.
func hashClientIPForFilter(secret, raw string) (hashed string, ok bool) {
	host, _, err := net.SplitHostPort(raw)
	if err != nil {
		host = raw
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}
	ip = ip.To16()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(ip)
	return hex.EncodeToString(mac.Sum(nil))[:16], true
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
