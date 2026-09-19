// SPDX-License-Identifier: GPL-3.0-or-later
package parser

import "testing"

func domains(entries []ParsedEntry) map[string]bool {
	m := map[string]bool{}
	for _, e := range entries {
		m[e.Domain] = true
	}
	return m
}

func TestDomainsParser(t *testing.T) {
	in := []byte(`# comment
! also a comment

ads.example.com
Tracker.EXAMPLE.org.
0.0.0.0 mixedhosts.example.net
bad domain with spaces
notadomain
malware.example.io  # inline comment
`)
	got := domains((&DomainsParser{}).mustParse(t, in))
	want := []string{"ads.example.com", "tracker.example.org", "mixedhosts.example.net", "malware.example.io"}
	for _, w := range want {
		if !got[w] {
			t.Errorf("expected %q to be parsed", w)
		}
	}
	if got["notadomain"] || got["bad domain with spaces"] {
		t.Error("non-domain lines should be dropped")
	}
	if len(got) != len(want) {
		t.Errorf("got %d domains, want %d: %v", len(got), len(want), got)
	}
}

func TestAdblockParser(t *testing.T) {
	in := []byte(`! Title: test
[Adblock Plus 2.0]
||ads.example.com^
||track.example.org^$third-party
||cdn.example.net/path
@@||allowed.example.com^
example.com##.banner
||plain.example.io
`)
	got := domains((&AdblockParser{}).mustParse(t, in))
	if !got["ads.example.com"] || !got["track.example.org"] || !got["cdn.example.net"] || !got["plain.example.io"] {
		t.Errorf("expected domain-anchored rules to parse, got %v", got)
	}
	if got["allowed.example.com"] {
		t.Error("exception rules (@@) must not be blocked")
	}
	if got["example.com"] {
		t.Error("cosmetic rules (##) must be skipped")
	}
}

// mustParse is a tiny helper so the tests read cleanly.
func (d *DomainsParser) mustParse(t *testing.T, b []byte) []ParsedEntry {
	t.Helper()
	e, err := d.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func (a *AdblockParser) mustParse(t *testing.T, b []byte) []ParsedEntry {
	t.Helper()
	e, err := a.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
