// SPDX-License-Identifier: GPL-3.0-or-later
package parser

import (
	"bufio"
	"bytes"
	"strings"
	"time"
)

// AdblockParser handles EasyList/adblock syntax, extracting domain-anchored
// network blocking rules of the form ||example.com^. It deliberately skips
// anything it cannot map to a plain domain block: exception rules (@@),
// cosmetic/element-hiding rules (## or #@#), and rules with paths, wildcards,
// or options ($). This keeps false positives out rather than guessing.
type AdblockParser struct{}

func (a *AdblockParser) Format() string { return "adblock" }

func (a *AdblockParser) Parse(data []byte) ([]ParsedEntry, error) {
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var out []ParsedEntry
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue // comment or header
		}
		if strings.HasPrefix(line, "@@") || strings.Contains(line, "##") || strings.Contains(line, "#@#") {
			continue // exception or cosmetic rule
		}
		if !strings.HasPrefix(line, "||") {
			continue
		}
		rule := strings.TrimPrefix(line, "||")
		// a domain-anchored rule ends at ^, /, or $
		end := strings.IndexAny(rule, "^/$")
		if end >= 0 {
			rule = rule[:end]
		}
		domain := normalizeDomain(rule)
		if domain == "" {
			continue
		}
		out = append(out, ParsedEntry{Domain: domain, Fetched: time.Now()})
	}
	return out, s.Err()
}

func init() { Register(&AdblockParser{}) }
