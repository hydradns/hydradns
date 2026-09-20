// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import (
	"net"

	"github.com/hydradns/hydra-core/internal/utils"
)

// anonymizeClientIP hashes clientIP for storage in the query log, when
// anonymization is enabled (see Engine.anonymizeClientIPs).
//
// clientIP is normally w.RemoteAddr().String(); for UDP/TCP that is always
// a "host:port" (or "[host]:port" for IPv6) string, never a bare IP, so the
// port must be stripped first: net.ParseIP (called inside
// utils.AnonymizeIP) rejects anything with a port suffix and would
// otherwise silently turn every stored row into an empty string. If the
// value doesn't parse as host:port (already a bare IP, or some
// unrecognized format), it's passed through unchanged and
// utils.AnonymizeIP applies its own defined fallback for unparseable
// input (returns "").
func anonymizeClientIP(clientIP string) string {
	return utils.AnonymizeIP(stripClientPort(clientIP))
}

// stripClientPort strips the port from a "host:port" or "[host]:port"
// address, returning the bare host. clientIP is normally
// w.RemoteAddr().String(), which for UDP/TCP always carries an ephemeral
// source port; storing that port defeats per-device filtering (every
// connection from the same device uses a different port, so two rows
// from one device look like two different clients) and clutters the
// dashboard's Client IP column. If addr does not parse as host:port
// (already a bare IP, or an unrecognized format) it is returned
// unchanged rather than mangled.
func stripClientPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
