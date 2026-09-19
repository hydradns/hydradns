// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"net"

	"github.com/hydradns/hydra-core/internal/utils"
)

// anonymizeClientIP hashes clientIP for storage in the query log, when
// anonymization is enabled (see Engine.anonymizeClientIPs).
//
// clientIP is normally w.RemoteAddr().String() — for UDP/TCP that is always
// a "host:port" (or "[host]:port" for IPv6) string, never a bare IP, so the
// port must be stripped first: net.ParseIP (called inside
// utils.AnonymizeIP) rejects anything with a port suffix and would
// otherwise silently turn every stored row into an empty string. If the
// value doesn't parse as host:port (already a bare IP, or some
// unrecognized format), it's passed through unchanged and
// utils.AnonymizeIP applies its own defined fallback for unparseable
// input (returns "").
func anonymizeClientIP(clientIP string) string {
	host := clientIP
	if h, _, err := net.SplitHostPort(clientIP); err == nil {
		host = h
	}
	return utils.AnonymizeIP(host)
}
