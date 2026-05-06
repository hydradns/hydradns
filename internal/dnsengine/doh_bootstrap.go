// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

// dohBootstrapHostnames is the curated set of hostnames that browsers,
// operating systems, and DoH-aware clients use to find their DNS-over-
// HTTPS or DNS-over-TLS endpoint. Resolving any of these is a strong
// signal that a client is about to bypass the local resolver entirely.
//
// We intercept the lookup at the engine level and return NXDOMAIN. The
// browser concludes its DoH provider is unreachable and falls back to
// the system resolver, which is HydraDNS. The user sees no error; their
// DNS just keeps flowing through us.
//
// This list is intentionally hardcoded, not user-editable, and not
// exposed through the /policies or /blocklists APIs. Treating it as
// invisible product-shipped behaviour means an end user who manages
// their own blocklists cannot accidentally remove it.
//
// Inclusion criteria:
//   - Hostname appears in a major browser's hardcoded DoH bootstrap
//     list (Chrome, Edge, Firefox, Brave) or in a major OS's encrypted-
//     DNS configuration (iOS, Android Private DNS).
//   - Hostname is publicly documented as a DoH/DoT endpoint by its
//     operator.
//
// To add a new entry: append to the source list below, regenerate the
// derived map, and add a regression test in doh_bootstrap_test.go that
// asserts the parent-domain walk still resolves the new entry.
//
// Last reviewed: 2026-05-06
var dohBootstrapHostnames = []string{
	// Cloudflare
	"cloudflare-dns.com",
	"mozilla.cloudflare-dns.com",
	"chrome.cloudflare-dns.com",
	"family.cloudflare-dns.com",
	"security.cloudflare-dns.com",
	"one.one.one.one",
	"1dot1dot1dot1.cloudflare-dns.com",

	// Google
	"dns.google",
	"dns.google.com",
	"dns64.dns.google",

	// Quad9
	"dns.quad9.net",
	"dns10.quad9.net",
	"dns11.quad9.net",

	// OpenDNS / Cisco
	"doh.opendns.com",
	"doh.familyshield.opendns.com",
	"doh.umbrella.com",

	// AdGuard
	"dns.adguard-dns.com",
	"dns-family.adguard-dns.com",
	"dns-unfiltered.adguard-dns.com",

	// NextDNS
	"dns.nextdns.io",

	// Mullvad
	"doh.mullvad.net",
	"dns.mullvad.net",

	// CleanBrowsing
	"doh.cleanbrowsing.org",

	// ControlD
	"dns.controld.com",
}

// dohBootstrapSet is a constant-time lookup of the hardcoded list above.
// Built once at package init; never mutated.
var dohBootstrapSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(dohBootstrapHostnames))
	for _, h := range dohBootstrapHostnames {
		m[h] = struct{}{}
	}
	return m
}()

// IsDoHBootstrap reports whether a query domain (already normalized)
// is one of the well-known DoH/DoT bootstrap hostnames, walking parent
// labels so a query for "chrome.cloudflare-dns.com" still hits
// "cloudflare-dns.com" in the set.
//
// The function is package-level (not a method on Engine) so unit tests
// can exercise it without spinning up a full engine.
func IsDoHBootstrap(domain string) bool {
	if domain == "" {
		return false
	}
	// Parent-domain walk. For a query "x.y.cloudflare-dns.com" we check
	// the full name, then "y.cloudflare-dns.com", then
	// "cloudflare-dns.com". We never check single-label names ("com")
	// because the set deliberately contains no TLDs.
	d := domain
	for {
		if _, ok := dohBootstrapSet[d]; ok {
			return true
		}
		i := indexByteFast(d, '.')
		if i < 0 {
			return false
		}
		d = d[i+1:]
		if d == "" {
			return false
		}
	}
}

// indexByteFast is a tiny strings.IndexByte clone kept here so the
// hot-path check above does not pay for an import-table lookup. The
// stdlib version is already an intrinsic on most platforms; this exists
// purely so the package stays import-light if the file is ever moved.
func indexByteFast(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
