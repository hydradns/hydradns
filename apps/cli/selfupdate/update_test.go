package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// releaseServer spins up an httptest server that mimics a release feed with a
// binary asset and a checksums file. If tamper is true, the published checksum
// does not match the served binary. downloadHits counts asset downloads.
func releaseServer(t *testing.T, tag, assetName string, binContent []byte, tamper bool) (*httptest.Server, *int32) {
	t.Helper()

	var downloadHits int32
	var srv *httptest.Server

	publishedSum := Sum256(binContent)
	if tamper {
		publishedSum = Sum256([]byte("something else entirely"))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/download/bin", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadHits, 1)
		w.Write(binContent)
	})
	mux.HandleFunc("/download/checksums", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", publishedSum, assetName)
	})
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		rel := Release{
			TagName: tag,
			Assets: []Asset{
				{Name: assetName, URL: srv.URL + "/download/bin"},
				{Name: "checksums.txt", URL: srv.URL + "/download/checksums"},
			},
		}
		json.NewEncoder(w).Encode(rel)
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &downloadHits
}

func TestLatestReleaseAndDownloadVerified(t *testing.T) {
	binContent := []byte("the new hydra binary bytes")
	assetName := "hydra_linux_amd64"
	srv, hits := releaseServer(t, "v1.0.1", assetName, binContent, false)

	u := New(Config{
		CurrentVersion: "1.0.0",
		FeedURL:        srv.URL + "/releases/latest",
		OS:             "linux",
		Arch:           "amd64",
	})

	rel, err := u.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Version() != "v1.0.1" {
		t.Fatalf("unexpected tag: %q", rel.Version())
	}

	newer, err := IsNewer(u.CurrentVersion(), rel.Version())
	if err != nil || !newer {
		t.Fatalf("expected v1.0.1 to be newer than 1.0.0 (got %v, err %v)", newer, err)
	}

	got, err := u.DownloadVerified(context.Background(), rel)
	if err != nil {
		t.Fatalf("DownloadVerified: %v", err)
	}
	if string(got) != string(binContent) {
		t.Fatalf("downloaded bytes mismatch: got %q", got)
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Fatalf("expected exactly one asset download, got %d", n)
	}
}

func TestDownloadVerifiedTamperedRefused(t *testing.T) {
	binContent := []byte("the new hydra binary bytes")
	assetName := "hydra_linux_amd64"
	srv, _ := releaseServer(t, "v1.0.1", assetName, binContent, true /* tamper */)

	u := New(Config{
		CurrentVersion: "1.0.0",
		FeedURL:        srv.URL + "/releases/latest",
		OS:             "linux",
		Arch:           "amd64",
	})

	rel, err := u.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}

	got, err := u.DownloadVerified(context.Background(), rel)
	if err == nil {
		t.Fatal("expected checksum mismatch to be refused, got nil error")
	}
	if got != nil {
		t.Fatalf("expected no bytes on refusal, got %d bytes", len(got))
	}
}

func TestApplyAtomicSwapAndBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydra")
	if err := os.WriteFile(target, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	backup, err := Apply([]byte("NEW BINARY"), target)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got, _ := os.ReadFile(target); string(got) != "NEW BINARY" {
		t.Errorf("target not updated: got %q", got)
	}
	if got, _ := os.ReadFile(backup); string(got) != "OLD BINARY" {
		t.Errorf("backup not preserved for rollback: got %q", got)
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("new binary is not executable: mode %v", fi.Mode())
	}
}

// TestDefaultFeedURLPointsAtMonorepo guards against DefaultFeedURL regressing
// to the archived hydradns/hydra-cli repo, which no longer receives releases
// (release.yml's release-cli job attaches binaries to hydradns/hydradns).
func TestDefaultFeedURLPointsAtMonorepo(t *testing.T) {
	const want = "https://api.github.com/repos/hydradns/hydradns/releases/latest"
	if DefaultFeedURL != want {
		t.Fatalf("DefaultFeedURL = %q, want %q", DefaultFeedURL, want)
	}
}

// realReleaseAssets mirrors exactly what a tagged hydradns/hydradns release
// carries once release.yml's release-cli matrix and release-cli-checksums job
// have both run: one hydra-<goos>-<goarch> binary per matrix leg (see
// .github/workflows/release.yml), a checksums.txt, and GitHub's own
// automatically attached source archives (present on every release,
// unrelated to any workflow step).
func realReleaseAssets() []Asset {
	names := []string{
		"hydra-linux-amd64",
		"hydra-linux-arm64",
		"hydra-darwin-amd64",
		"hydra-darwin-arm64",
		"checksums.txt",
		"Source code (zip)",
		"Source code (tar.gz)",
	}
	assets := make([]Asset, len(names))
	for i, n := range names {
		assets[i] = Asset{Name: n, URL: "https://example.invalid/" + n}
	}
	return assets
}

// TestSelectAssetAgainstRealReleaseLayout checks that every OS/Arch pair the
// release-cli matrix actually builds resolves to exactly its own binary, with
// no cross-match between e.g. linux/amd64 and darwin/amd64, and that neither
// checksums.txt nor GitHub's auto-attached source archives are ever selected
// as a binary.
func TestSelectAssetAgainstRealReleaseLayout(t *testing.T) {
	rel := &Release{TagName: "v0.1.0", Assets: realReleaseAssets()}

	cases := []struct {
		os, arch, want string
	}{
		{"linux", "amd64", "hydra-linux-amd64"},
		{"linux", "arm64", "hydra-linux-arm64"},
		{"darwin", "amd64", "hydra-darwin-amd64"},
		{"darwin", "arm64", "hydra-darwin-arm64"},
	}
	for _, tc := range cases {
		u := New(Config{OS: tc.os, Arch: tc.arch})
		got, err := u.SelectAsset(rel)
		if err != nil {
			t.Errorf("SelectAsset(%s/%s): %v", tc.os, tc.arch, err)
			continue
		}
		if got.Name != tc.want {
			t.Errorf("SelectAsset(%s/%s) = %q, want %q", tc.os, tc.arch, got.Name, tc.want)
		}
	}
}

// TestSelectAssetNoCLIAssetsIsClear checks that a release with no CLI
// binaries attached (e.g. a partial or non-CLI release on the shared
// hydradns/hydradns feed) fails with a message that says so, rather than a
// bare "not found".
func TestSelectAssetNoCLIAssetsIsClear(t *testing.T) {
	rel := &Release{
		TagName: "v0.2.0",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "https://example.invalid/checksums.txt"},
			{Name: "Source code (zip)", URL: "https://example.invalid/src.zip"},
		},
	}
	u := New(Config{OS: "linux", Arch: "amd64"})
	_, err := u.SelectAsset(rel)
	if err == nil {
		t.Fatal("expected an error when no CLI asset is attached to the release")
	}
	if !strings.Contains(err.Error(), "may not include CLI binaries") {
		t.Fatalf("error should explain the release may lack CLI binaries, got: %v", err)
	}
}

// TestSelectChecksumsAgainstRealReleaseLayout checks that the checksums asset
// is found (and not, e.g., accidentally shadowed by a binary name).
func TestSelectChecksumsAgainstRealReleaseLayout(t *testing.T) {
	rel := &Release{TagName: "v0.1.0", Assets: realReleaseAssets()}
	u := New(Config{OS: "linux", Arch: "amd64"})
	got, err := u.SelectChecksums(rel)
	if err != nil {
		t.Fatalf("SelectChecksums: %v", err)
	}
	if got.Name != "checksums.txt" {
		t.Fatalf("SelectChecksums returned %q, want checksums.txt", got.Name)
	}
}

func TestApplyRefusesEmpty(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "hydra")
	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(nil, target); err == nil {
		t.Fatal("Apply should refuse an empty binary")
	}
	// The original binary must be untouched after a refusal.
	if got, _ := os.ReadFile(target); string(got) != "OLD" {
		t.Errorf("target modified on refusal: got %q", got)
	}
}
