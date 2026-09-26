// SPDX-License-Identifier: Apache-2.0
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hydradns/hydradns/apps/core/internal/logger"
)

// anonSecretPlaceholder is the documented "unset" value for
// dataplane.anonymization.secret. Any install that doesn't override it
// (via config or HYDRA_ANON_SECRET) gets a generated, per-install secret
// instead of sharing this well-known string as an HMAC key.
const anonSecretPlaceholder = "change-me-in-production"

// anonSecretFileName is the file the generated secret is persisted to,
// alongside the SQLite DB, so it survives restarts.
const anonSecretFileName = "anon_secret"

// anonSecretByteLen is the amount of crypto/rand entropy generated for a
// fresh secret (hex-encoded on disk, so the file is 2x this many bytes).
const anonSecretByteLen = 32

// ResolveAnonymizationSecret returns the HMAC key utils.AnonymizeIP will
// use to hash client IPs before they're written to the query log, when
// anonymization is enabled. Callers should only invoke this (and only call
// utils.InitSecret with the result) when
// config.DefaultConfig.DataPlane.Anonymization.Enabled is true: resolving
// or generating a secret that nothing will ever use is harmless but
// surprising (a mystery file appearing in the data dir on every install).
//
// Priority:
//  1. HYDRA_ANON_SECRET env var, if set to a real (non-placeholder) value.
//  2. cfgValue (dataplane.anonymization.secret from config.yaml), if it's a
//     real value.
//  3. A secret generated on first boot with crypto/rand and persisted at
//     <dataDir>/anon_secret (0600), reused on every later boot. dataDir is
//     normally the directory holding the SQLite DB. Creation is
//     O_EXCL-guarded so concurrent loaders (e.g. controlplane + dataplane
//     starting in the same container) converge on one value instead of
//     racing.
//  4. If dataDir isn't writable, a random in-memory secret, logged as a
//     warning since hashes won't survive a restart. The secret value
//     itself is never logged.
func ResolveAnonymizationSecret(cfgValue, dataDir string) string {
	if v := realSecretValue(os.Getenv("HYDRA_ANON_SECRET")); v != "" {
		return v
	}
	if v := realSecretValue(cfgValue); v != "" {
		return v
	}

	secret, err := loadOrCreatePersistedSecret(dataDir)
	if err != nil {
		fallback, genErr := generateSecretHex()
		if genErr != nil {
			// crypto/rand is broken: every install would otherwise share
			// the same hardcoded "unavailable-anon-secret" HMAC key,
			// which defeats anonymization for all of them at once.
			// crypto/rand failure is effectively fatal for anything else
			// that needs randomness too, so fail closed here rather than
			// silently handing out a known key.
			FatalFunc("anonymization: failed to generate a secret (crypto/rand: %v) — refusing to start with HYDRA_ANONYMIZE_CLIENT_IPS enabled rather than use a shared fallback key", genErr)
			return "" // unreachable when FatalFunc actually exits; keeps a test-overridden FatalFunc from continuing with a bogus secret
		}
		logger.Log.Warnf("anonymization: could not persist secret under %s (%v); using an in-memory secret for this run — hashed client IPs will change on every restart", dataDir, err)
		return fallback
	}
	return secret
}

// realSecretValue returns v if it looks like an operator-supplied secret,
// or "" if it's empty, the documented placeholder, or an unexpanded
// shell-style default (e.g. a stale "${HYDRA_ANON_SECRET:-...}" left in
// config.yaml; the app's YAML loader does not expand that syntax, so it
// must never be used verbatim as key material).
func realSecretValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == anonSecretPlaceholder || strings.Contains(v, "${") {
		return ""
	}
	return v
}

func generateSecretHex() (string, error) {
	b := make([]byte, anonSecretByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// loadOrCreatePersistedSecret reads <dataDir>/anon_secret, generating and
// persisting it on first use. Safe for concurrent callers (in-process or
// separate processes sharing dataDir).
func loadOrCreatePersistedSecret(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dataDir, anonSecretFileName)

	// Fast path: another boot (or another process, right now) already
	// wrote it.
	if s, err := readSecretFile(path); err == nil && s != "" {
		return s, nil
	}

	generated, err := generateSecretHex()
	if err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			// Lost the race to create it: wait for the winner to finish
			// writing and read back what they wrote.
			return waitForPersistedSecret(path)
		}
		return "", fmt.Errorf("create secret file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			logger.Log.Warnf("anonymization: failed to close secret file %s: %v", path, cerr)
		}
	}()

	if _, err := f.WriteString(generated); err != nil {
		return "", fmt.Errorf("write secret file: %w", err)
	}
	// Belt-and-suspenders: force 0600 regardless of umask.
	if err := f.Chmod(0o600); err != nil {
		return "", fmt.Errorf("chmod secret file: %w", err)
	}
	return generated, nil
}

func readSecretFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// waitForPersistedSecret polls for a secret file that another loader (in
// this process or another) is in the middle of creating.
func waitForPersistedSecret(path string) (string, error) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if s, err := readSecretFile(path); err == nil && s != "" {
			return s, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timed out waiting for concurrent writer to finish %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
