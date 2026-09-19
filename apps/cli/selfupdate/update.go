package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultFeedURL is the GitHub "latest release" REST endpoint for hydra-cli. It
// returns a JSON document matching the Release type below. Override it with the
// HYDRA_UPDATE_URL env var or the --url flag to point at a private feed.
const DefaultFeedURL = "https://api.github.com/repos/hydradns/hydra-cli/releases/latest"

// maxDownloadBytes caps any single download to guard against a hostile feed.
const maxDownloadBytes = 512 << 20 // 512 MiB

// Asset is a downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Release is the subset of a GitHub release document we consume.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Version returns the release's version string (its tag).
func (r *Release) Version() string { return r.TagName }

// Config configures an Updater. Zero values are filled with sensible defaults.
type Config struct {
	// CurrentVersion is the built-in version of the running binary.
	CurrentVersion string
	// FeedURL is the release feed to query. Defaults to DefaultFeedURL.
	FeedURL string
	// HTTPClient is used for all requests. Defaults to a 30s-timeout client.
	HTTPClient *http.Client
	// OS and Arch select the matching release asset. Default to runtime values.
	OS   string
	Arch string
	// AssetName, if set, forces selection of the asset with this exact name.
	AssetName string
}

// Updater checks a release feed and downloads/verifies release assets.
type Updater struct {
	cfg    Config
	client *http.Client
}

// New builds an Updater, filling defaults for any unset Config fields.
func New(cfg Config) *Updater {
	if cfg.FeedURL == "" {
		cfg.FeedURL = DefaultFeedURL
	}
	if cfg.OS == "" {
		cfg.OS = runtime.GOOS
	}
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Updater{cfg: cfg, client: client}
}

// CurrentVersion returns the configured current version.
func (u *Updater) CurrentVersion() string { return u.cfg.CurrentVersion }

// LatestRelease fetches and decodes the latest release from the feed.
func (u *Updater) LatestRelease(ctx context.Context) (*Release, error) {
	body, err := u.get(ctx, u.cfg.FeedURL)
	if err != nil {
		return nil, err
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("invalid release feed: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release feed has no tag_name")
	}
	return &rel, nil
}

// SelectAsset returns the release asset matching the configured OS/Arch (or the
// exact AssetName override), skipping the checksums file.
func (u *Updater) SelectAsset(rel *Release) (*Asset, error) {
	if u.cfg.AssetName != "" {
		for i := range rel.Assets {
			if rel.Assets[i].Name == u.cfg.AssetName {
				return &rel.Assets[i], nil
			}
		}
		return nil, fmt.Errorf("asset %q not found in release %s", u.cfg.AssetName, rel.TagName)
	}

	osTok := strings.ToLower(u.cfg.OS)
	archTok := strings.ToLower(u.cfg.Arch)
	for i := range rel.Assets {
		n := strings.ToLower(rel.Assets[i].Name)
		if isChecksumName(n) {
			continue
		}
		if strings.Contains(n, osTok) && strings.Contains(n, archTok) {
			return &rel.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("no asset for %s/%s in release %s", u.cfg.OS, u.cfg.Arch, rel.TagName)
}

// SelectChecksums returns the published checksums asset for the release.
func (u *Updater) SelectChecksums(rel *Release) (*Asset, error) {
	for i := range rel.Assets {
		if isChecksumName(rel.Assets[i].Name) {
			return &rel.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("no checksums file in release %s", rel.TagName)
}

func isChecksumName(name string) bool {
	n := strings.ToLower(name)
	return n == "checksums.txt" ||
		n == "sha256sums.txt" ||
		n == "sha256sums" ||
		strings.Contains(n, "checksum") ||
		strings.Contains(n, "sha256sum")
}

// DownloadVerified downloads the OS/Arch asset and its published checksum,
// verifies the asset against that checksum, and returns the verified bytes. It
// never touches the running binary — callers pass the result to Apply. A
// checksum mismatch (tampered or corrupt download) returns an error and no data.
func (u *Updater) DownloadVerified(ctx context.Context, rel *Release) ([]byte, error) {
	asset, err := u.SelectAsset(rel)
	if err != nil {
		return nil, err
	}
	sumsAsset, err := u.SelectChecksums(rel)
	if err != nil {
		return nil, err
	}

	sumData, err := u.get(ctx, sumsAsset.URL)
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	checksums := ParseChecksums(sumData)
	want, ok := checksums[asset.Name]
	if !ok {
		return nil, fmt.Errorf("no checksum published for asset %q", asset.Name)
	}

	binData, err := u.get(ctx, asset.URL)
	if err != nil {
		return nil, fmt.Errorf("download asset: %w", err)
	}
	if err := VerifyChecksum(binData, want); err != nil {
		return nil, fmt.Errorf("refusing update, %w", err)
	}
	return binData, nil
}

func (u *Updater) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// GitHub's API requires a User-Agent; harmless for other feeds.
	req.Header.Set("User-Agent", "hydra-cli-selfupdate")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes))
}

// Apply atomically replaces the binary at targetPath with newBinary. It writes
// to a temp file in the same directory, fsyncs it, chmods it to the existing
// binary's mode, moves the current binary aside to "<targetPath>.old" for
// rollback, then renames the temp file into place. Because rename within a
// directory is atomic, there is never a partially written binary at targetPath.
// The checksum MUST already be verified before calling this. It returns the path
// of the preserved backup.
func Apply(newBinary []byte, targetPath string) (string, error) {
	if len(newBinary) == 0 {
		return "", fmt.Errorf("refusing to install empty binary")
	}

	dir := filepath.Dir(targetPath)

	// Preserve the existing binary's mode; default to 0755 for a fresh install.
	var mode os.FileMode = 0o755
	if fi, err := os.Stat(targetPath); err == nil {
		if p := fi.Mode().Perm(); p != 0 {
			mode = p
		}
	}

	tmp, err := os.CreateTemp(dir, ".hydra-update-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("sync temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp binary: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return "", fmt.Errorf("chmod temp binary: %w", err)
	}

	backup := targetPath + ".old"
	// Move the current binary aside for rollback (if one exists).
	if _, err := os.Stat(targetPath); err == nil {
		_ = os.Remove(backup) // clear any stale backup
		if err := os.Rename(targetPath, backup); err != nil {
			return "", fmt.Errorf("backup current binary: %w", err)
		}
	}

	// Atomic swap of the new binary into place.
	if err := os.Rename(tmpName, targetPath); err != nil {
		// Roll back: restore the previous binary if we moved it aside.
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, targetPath)
		}
		return "", fmt.Errorf("install new binary: %w", err)
	}

	success = true
	return backup, nil
}
