package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ParseChecksums parses a checksums file in the common "sha256sum" format, where
// each line is "<hex-digest>  <filename>". A leading '*' on the filename (binary
// mode) and blank/comment lines are handled. It returns a map of filename to the
// lowercase hex digest.
func ParseChecksums(data []byte) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sum := strings.ToLower(fields[0])
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		out[name] = sum
	}
	return out
}

// Sum256 returns the lowercase hex-encoded SHA-256 digest of data.
func Sum256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// VerifyChecksum returns nil only if the SHA-256 of data equals expectedHex
// (compared case-insensitively). Any mismatch, or an empty expected value, is an
// error so callers can refuse to install tampered or unverifiable downloads.
func VerifyChecksum(data []byte, expectedHex string) error {
	want := strings.ToLower(strings.TrimSpace(expectedHex))
	if want == "" {
		return fmt.Errorf("no expected checksum provided")
	}
	got := Sum256(data)
	if got != want {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", want, got)
	}
	return nil
}
