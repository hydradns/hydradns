// SPDX-License-Identifier: GPL-3.0-or-later
package dnsengine

import "testing"

func TestIsDoHBootstrap(t *testing.T) {
	cases := []struct {
		domain string
		want   bool
		note   string
	}{
		// Direct hits — exact strings from the curated list.
		{"dns.google", true, "Google DoH endpoint"},
		{"cloudflare-dns.com", true, "Cloudflare apex"},
		{"chrome.cloudflare-dns.com", true, "Chrome's DoH endpoint"},
		{"dns.quad9.net", true, "Quad9"},
		{"doh.opendns.com", true, "OpenDNS"},
		{"dns.adguard-dns.com", true, "AdGuard"},
		{"dns.nextdns.io", true, "NextDNS"},

		// Parent-domain walk: a sub-subdomain still resolves to a
		// listed parent.
		{"sub.chrome.cloudflare-dns.com", true, "deep subdomain of listed parent"},
		{"a.b.c.dns.google", true, "deep subdomain of dns.google"},

		// Negatives — domains that look related but are not in the
		// list. These must not match, otherwise the curated guarantee
		// is broken.
		{"google.com", false, "real google.com is unrelated to DoH"},
		{"cloudflare.com", false, "Cloudflare apex is not the DoH apex"},
		{"www.google.com", false, "regular Google subdomain"},
		{"mail.google.com", false, "Gmail subdomain"},
		{"facebook.com", false, "completely unrelated"},
		{"", false, "empty domain"},
		{"com", false, "TLD never matches"},

		// Confirm tail handling: if a query happens to share only the
		// suffix label ("google") with a listed name ("dns.google"),
		// it must not match.
		{"google.local", false, "suffix-of-suffix should not match"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.note, func(t *testing.T) {
			got := IsDoHBootstrap(tc.domain)
			if got != tc.want {
				t.Errorf("IsDoHBootstrap(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}

// TestIsDoHBootstrap_ListIntegrity guards against accidental drift in
// the curated list (e.g. a duplicate entry, an empty string, or a name
// that does not look like a DNS hostname).
func TestIsDoHBootstrap_ListIntegrity(t *testing.T) {
	if len(dohBootstrapHostnames) == 0 {
		t.Fatal("DoH bootstrap list is empty")
	}
	seen := make(map[string]bool, len(dohBootstrapHostnames))
	for _, h := range dohBootstrapHostnames {
		if h == "" {
			t.Errorf("empty entry in dohBootstrapHostnames")
		}
		if seen[h] {
			t.Errorf("duplicate entry: %q", h)
		}
		if !contains(h, ".") {
			t.Errorf("entry has no dot, looks like a TLD: %q", h)
		}
		seen[h] = true
	}
	if len(dohBootstrapSet) != len(dohBootstrapHostnames) {
		t.Errorf("set size %d != list size %d (duplicates likely)",
			len(dohBootstrapSet), len(dohBootstrapHostnames))
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
