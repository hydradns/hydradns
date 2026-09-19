package selfupdate

import (
	"strings"
	"testing"
)

func TestVerifyChecksumGood(t *testing.T) {
	data := []byte("hydra binary v1.0.1")
	sum := Sum256(data)

	if err := VerifyChecksum(data, sum); err != nil {
		t.Fatalf("good checksum should verify: %v", err)
	}
	// Digest comparison is case-insensitive.
	if err := VerifyChecksum(data, strings.ToUpper(sum)); err != nil {
		t.Fatalf("uppercase checksum should verify: %v", err)
	}
}

func TestVerifyChecksumTampered(t *testing.T) {
	data := []byte("hydra binary v1.0.1")
	// A checksum computed over different content must be refused.
	wrong := Sum256([]byte("tampered content"))
	if err := VerifyChecksum(data, wrong); err == nil {
		t.Fatal("tampered checksum must be refused, got nil error")
	}
	// An empty expected checksum must also be refused.
	if err := VerifyChecksum(data, ""); err == nil {
		t.Fatal("empty checksum must be refused, got nil error")
	}
}

func TestParseChecksums(t *testing.T) {
	file := "ABC123  hydra_linux_amd64\n" +
		"def456 *hydra_darwin_arm64\n" +
		"# a comment line\n" +
		"\n" +
		"garbage-no-filename\n"

	m := ParseChecksums([]byte(file))
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(m), m)
	}
	if got := m["hydra_linux_amd64"]; got != "abc123" {
		t.Errorf("linux digest lowercased wrong: got %q", got)
	}
	if got := m["hydra_darwin_arm64"]; got != "def456" {
		t.Errorf("darwin digest (leading * stripped) wrong: got %q", got)
	}
}
