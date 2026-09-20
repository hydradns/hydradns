package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
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

// TestParseChecksumsMatchesRealSha256sumOutput proves that ParseChecksums
// accepts the exact file release.yml's release-cli-checksums job produces:
// `sha256sum hydra-* > checksums.txt`, run over real files, piped straight
// into this package's own parser. This is the same format/tooling combo used
// in CI (ubuntu-latest), so a mismatch here would mean `hydra update` can
// never verify a real release asset.
func TestParseChecksumsMatchesRealSha256sumOutput(t *testing.T) {
	sha256sumPath, err := exec.LookPath("sha256sum")
	if err != nil {
		t.Skip("sha256sum not available on this machine (CI runs on ubuntu-latest, where it is)")
	}

	dir := t.TempDir()
	files := map[string][]byte{
		"hydra-linux-amd64":  []byte("dummy linux/amd64 binary contents"),
		"hydra-darwin-arm64": []byte("dummy darwin/arm64 binary contents"),
	}
	names := make([]string, 0, len(files))
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		names = append(names, name)
	}

	// Mirrors the workflow step exactly: `sha256sum hydra-* > checksums.txt`.
	args := append([]string{}, names...)
	cmd := exec.Command(sha256sumPath, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("sha256sum: %v", err)
	}

	got := ParseChecksums(out)
	if len(got) != len(files) {
		t.Fatalf("expected %d entries, got %d: %v", len(files), len(got), got)
	}
	for name, content := range files {
		want := Sum256(content)
		sum, ok := got[name]
		if !ok {
			t.Errorf("no checksum parsed for %s (raw sha256sum output: %q)", name, out)
			continue
		}
		if sum != want {
			t.Errorf("%s: parsed checksum %q does not match Sum256 %q", name, sum, want)
		}
		if err := VerifyChecksum(content, sum); err != nil {
			t.Errorf("%s: VerifyChecksum failed against parsed checksum: %v", name, err)
		}
	}
}
