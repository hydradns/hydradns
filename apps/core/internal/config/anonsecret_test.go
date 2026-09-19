// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestResolveAnonymizationSecret_EnvValueRespectedUnchanged(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "operator-supplied-secret")
	dir := t.TempDir()

	got := ResolveAnonymizationSecret("change-me-in-production", dir)
	if got != "operator-supplied-secret" {
		t.Fatalf("got %q, want env value used unchanged", got)
	}
	if _, err := os.Stat(filepath.Join(dir, anonSecretFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected no secret file to be created when an explicit secret is supplied, stat err=%v", err)
	}
}

func TestResolveAnonymizationSecret_ConfigValueRespectedUnchanged(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	got := ResolveAnonymizationSecret("configured-real-secret", dir)
	if got != "configured-real-secret" {
		t.Fatalf("got %q, want config value used unchanged", got)
	}
}

func TestResolveAnonymizationSecret_PlaceholderTriggersGeneration(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	got := ResolveAnonymizationSecret(anonSecretPlaceholder, dir)
	if got == "" || got == anonSecretPlaceholder {
		t.Fatalf("expected a freshly generated secret, got %q", got)
	}

	path := filepath.Join(dir, anonSecretFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected secret to be persisted at %s: %v", path, err)
	}
	if strings.TrimSpace(string(b)) != got {
		t.Fatalf("persisted secret %q does not match returned secret %q", b, got)
	}
}

func TestResolveAnonymizationSecret_EmptyConfigValueTriggersGeneration(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	got := ResolveAnonymizationSecret("", dir)
	if got == "" {
		t.Fatalf("expected a generated secret for an empty config value")
	}
}

func TestResolveAnonymizationSecret_PersistsAcrossTwoLoads(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	first := ResolveAnonymizationSecret(anonSecretPlaceholder, dir)
	second := ResolveAnonymizationSecret(anonSecretPlaceholder, dir)
	if first != second {
		t.Fatalf("secret changed across two loads of the same data dir: %q != %q", first, second)
	}
}

func TestResolveAnonymizationSecret_ConcurrentLoadersAgree(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	const n = 16
	results := make([]string, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = ResolveAnonymizationSecret(anonSecretPlaceholder, dir)
		}(i)
	}
	wg.Wait()

	for i := 1; i < n; i++ {
		if results[i] == "" {
			t.Fatalf("loader %d returned an empty secret", i)
		}
		if results[i] != results[0] {
			t.Fatalf("concurrent loaders disagreed: loader 0 got %q, loader %d got %q", results[0], i, results[i])
		}
	}
}

func TestResolveAnonymizationSecret_FileModeIsOwnerReadWriteOnly(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	dir := t.TempDir()

	ResolveAnonymizationSecret(anonSecretPlaceholder, dir)

	info, err := os.Stat(filepath.Join(dir, anonSecretFileName))
	if err != nil {
		t.Fatalf("stat secret file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("secret file mode = %o, want 0600", perm)
	}
}

func TestResolveAnonymizationSecret_UnwritableDataDirFallsBackWithoutPanic(t *testing.T) {
	t.Setenv("HYDRA_ANON_SECRET", "")
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions don't block writes")
	}

	base := t.TempDir()
	roParent := filepath.Join(base, "readonly-parent")
	if err := os.MkdirAll(roParent, 0o500); err != nil {
		t.Fatalf("test setup: %v", err)
	}
	// A data dir that doesn't exist yet, under a parent we can't write to —
	// MkdirAll must fail here.
	dataDir := filepath.Join(roParent, "data")

	var got string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ResolveAnonymizationSecret panicked: %v", r)
			}
		}()
		got = ResolveAnonymizationSecret(anonSecretPlaceholder, dataDir)
	}()

	if got == "" {
		t.Fatalf("expected a fallback in-memory secret, got empty string")
	}
	if _, err := os.Stat(filepath.Join(dataDir, anonSecretFileName)); !os.IsNotExist(err) {
		t.Fatalf("expected no secret file to be persisted under an unwritable dir, stat err=%v", err)
	}
}

func TestResolveAnonymizationSecret_UnwritableDirFallbackVariesAcrossCalls(t *testing.T) {
	// Sanity check that the fallback is actually random and not a fixed
	// hardcoded string (which would recreate the "shared secret" bug).
	t.Setenv("HYDRA_ANON_SECRET", "")
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions don't block writes")
	}

	base := t.TempDir()
	roParent := filepath.Join(base, "readonly-parent")
	if err := os.MkdirAll(roParent, 0o500); err != nil {
		t.Fatalf("test setup: %v", err)
	}
	dataDir := filepath.Join(roParent, "data")

	a := ResolveAnonymizationSecret(anonSecretPlaceholder, dataDir)
	b := ResolveAnonymizationSecret(anonSecretPlaceholder, dataDir)
	if a == b {
		t.Fatalf("expected fallback secrets to differ across calls (no persistence possible), both were %q", a)
	}
}
