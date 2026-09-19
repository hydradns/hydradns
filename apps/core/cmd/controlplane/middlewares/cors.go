// SPDX-License-Identifier: GPL-3.0-or-later
package middlewares

import (
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a gin middleware that applies CORS configuration.
//
// CORS_ORIGINS (comma-separated) is the explicit allowlist, checked first.
// Defaults to the dashboard's own default origin (localhost:3000) for
// development. Setting it to "*" allows any website to call the API — a
// startup warning is logged, but it still works, for operators who
// understand the tradeoff (e.g. a fully isolated dev environment).
//
// On top of the explicit allowlist, an automatic same-host rule is applied
// so a dashboard reached over the LAN works without editing CORS_ORIGINS.
// Disable it with CORS_ALLOW_SAME_HOST=false. See sameHostOriginAllowed for
// exactly what it allows and why it is safe against DNS rebinding.
func CORS() gin.HandlerFunc {
	origins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		origins = strings.Split(env, ",")
	}
	for _, o := range origins {
		if o == "*" {
			log.Printf("WARNING: CORS_ORIGINS=* — any website can call this API. Only use this for local development or if you understand the risk.")
			break
		}
	}

	cfg := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// CORS_ALLOW_SAME_HOST defaults to on; only the literal value "false"
	// turns it off, matching the design doc's "disable with
	// CORS_ALLOW_SAME_HOST=false" wording.
	if os.Getenv("CORS_ALLOW_SAME_HOST") != "false" {
		cfg.AllowOriginWithContextFunc = sameHostOriginAllowed
	}

	return cors.New(cfg)
}

// sameHostOriginAllowed is the automatic same-host CORS rule.
//
// gin-contrib/cors (v1.7.6, config.go:118-124, isOriginValid) only calls
// AllowOriginWithContextFunc as a fallback, after the explicit CORS_ORIGINS
// allowlist (validateOrigin) has already failed to match — so this never
// widens access beyond the explicit list plus what's granted below, and it
// never overrides an explicit "*" (AllowAllOrigins short-circuits to true
// before the fallback is reached). It also runs for preflight OPTIONS
// requests: applyCors (config.go:71-100) resolves the origin before
// branching on request method, so the same check governs both preflight
// and the actual request.
//
// The request's Origin is allowed when all of:
//
//	(a) the Origin's hostname equals the hostname in the request's Host
//	    header — the API's own address as the browser reached it — with
//	    ports ignored. This is what lets a dashboard opened at
//	    http://192.168.1.53:3000 call the API at http://192.168.1.53:8080
//	    with zero configuration: same host, different port.
//	(b) that hostname is an IP literal (IPv4 or IPv6) or exactly
//	    "localhost". DNS rebinding: an attacker page at
//	    http://evil.example:3000 can rebind evil.example's DNS record to
//	    the appliance's IP, and then both the Origin's host and the Host
//	    header the browser sends would read "evil.example" — a same-host
//	    check with no further restriction would pass. Restricting the
//	    automatic rule to IP literals and localhost defeats that, because
//	    a rebinding attack necessarily routes through an attacker-
//	    controlled DNS name, and DNS names are exactly what this check
//	    excludes. Named hosts (pi.local, hydra.lan, a reverse-proxy
//	    domain) are never covered by this rule and need an explicit
//	    CORS_ORIGINS entry.
//	(c) the Origin's scheme is http or https.
//
// Host header trust: c.Request.Host is Go's net/http parse of the literal
// Host header (or :authority on HTTP/2) the client sent to this server —
// never X-Forwarded-Host or any other proxy-supplied header, which a client
// sitting in front of a reverse proxy could set to anything. A reverse
// proxy in front of this API must forward the real Host unchanged, or this
// rule stops firing for the LAN case it exists to cover (the operator then
// needs an explicit CORS_ORIGINS entry for the proxy's hostname anyway).
//
// A non-browser client (curl, a script) can send any Origin and any Host
// header it wants; nothing here stops it, and that's fine. CORS is a
// restriction browsers place on what a web page can make the visitor's own
// browser do to a third-party API — it does not and cannot constrain a
// client that isn't a browser. A non-browser client that already holds a
// valid bearer token can call this API directly regardless of Origin or
// Host; spoofing Host in the same-host check gains it nothing it didn't
// already have.
func sameHostOriginAllowed(c *gin.Context, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	originHost := u.Hostname()
	if originHost == "" {
		return false
	}

	requestHost := hostnameOnly(c.Request.Host)
	if requestHost == "" || !strings.EqualFold(originHost, requestHost) {
		return false
	}

	if strings.EqualFold(requestHost, "localhost") {
		return true
	}
	return net.ParseIP(requestHost) != nil
}

// hostnameOnly strips an optional port from a Host-header-style string,
// including bracketed IPv6 literals (e.g. "[::1]:8080" -> "::1").
func hostnameOnly(hostport string) string {
	if hostport == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	// No port present (net.SplitHostPort failed because there's no ":port"
	// suffix) — hostport is already just the host. Still strip stray IPv6
	// brackets for the rare case of a bracketed literal with no port.
	return strings.Trim(hostport, "[]")
}
