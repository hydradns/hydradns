// Package selfupdate implements checking a release feed for a newer hydra
// release, verifying the downloaded asset against a published checksum, and
// atomically replacing the running binary with a backup kept for rollback.
package selfupdate

import (
	"fmt"
	"strconv"
	"strings"
)

// parseVersion parses a version like "v1.2.3", "1.2", or "1.2.3-rc1" into a
// [major, minor, patch] tuple. A leading "v"/"V" is tolerated. Any pre-release
// or build metadata (after '-' or '+') is ignored for this first-cut numeric
// comparison. Missing components default to 0 (e.g. "1.2" == "1.2.0").
func parseVersion(s string) ([3]int, error) {
	v := strings.TrimSpace(s)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return [3]int{}, fmt.Errorf("empty version string")
	}

	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return [3]int{}, fmt.Errorf("invalid version %q: too many components", s)
	}

	var out [3]int
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("invalid version %q: bad component %q", s, parts[i])
		}
		out[i] = n
	}
	return out, nil
}

// Compare returns -1 if a < b, 0 if a == b, and 1 if a > b, comparing the
// numeric major.minor.patch cores.
func Compare(a, b string) (int, error) {
	av, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		switch {
		case av[i] < bv[i]:
			return -1, nil
		case av[i] > bv[i]:
			return 1, nil
		}
	}
	return 0, nil
}

// IsNewer reports whether candidate is a strictly newer version than current.
func IsNewer(current, candidate string) (bool, error) {
	c, err := Compare(candidate, current)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}
