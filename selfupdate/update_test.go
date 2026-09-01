package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
