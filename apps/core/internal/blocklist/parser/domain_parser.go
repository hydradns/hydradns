// SPDX-License-Identifier: Apache-2.0
package parser

import (
	"bufio"
	"bytes"
	"strings"
	"time"
)

// DomainsParser handles plain domain-per-line lists (one hostname per line),
// the format used by OISD, HaGeZi and many modern blocklists. Comment lines
// (# or !) and blank lines are skipped. A leading 0.0.0.0/127.0.0.1 and any
// inline comment are tolerated so near-hosts lists still parse.
type DomainsParser struct{}

func (d *DomainsParser) Format() string { return "domains" }

func (d *DomainsParser) Parse(data []byte) ([]ParsedEntry, error) {
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []ParsedEntry
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		// drop an inline comment
		if i := strings.IndexAny(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		// tolerate a leading IP (0.0.0.0 domain / 127.0.0.1 domain)
		cand := fields[0]
		if len(fields) >= 2 && (cand == "0.0.0.0" || cand == "127.0.0.1" || cand == "::1") {
			cand = fields[1]
		}
		domain := normalizeDomain(cand)
		if domain == "" {
			continue
		}
		out = append(out, ParsedEntry{Domain: domain, Fetched: time.Now()})
	}
	return out, s.Err()
}

// normalizeDomain lowercases, strips a trailing dot, and rejects anything that
// is not a bare hostname (contains a space, slash, or has no dot).
func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(d, ".")))
	if d == "" || strings.ContainsAny(d, " /\\*") || !strings.Contains(d, ".") {
		return ""
	}
	return d
}

func init() { Register(&DomainsParser{}) }
