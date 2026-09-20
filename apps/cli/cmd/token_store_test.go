package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	// os.UserHomeDir on Windows reads USERPROFILE, not HOME.
	t.Setenv("USERPROFILE", home)
	return home
}

func TestSaveToken_FixesLooseFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	home := withTempHome(t)
	dir := filepath.Join(home, ".hydra")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenPath, []byte("stale-token\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := saveToken("fresh-token")
	if err != nil {
		t.Fatalf("saveToken: %v", err)
	}
	if got != tokenPath {
		t.Fatalf("saveToken returned %q, want %q", got, tokenPath)
	}

	fi, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("token file perm = %o, want 0600", perm)
	}

	b, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "fresh-token\n" {
		t.Errorf("token content = %q, want %q", b, "fresh-token\n")
	}
}

func TestSaveToken_FixesLooseDirPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	home := withTempHome(t)
	dir := filepath.Join(home, ".hydra")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := saveToken("fresh-token"); err != nil {
		t.Fatalf("saveToken: %v", err)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}
}

func TestSaveToken_RefusesSymlinkedTokenPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows by default")
	}
	home := withTempHome(t)
	dir := filepath.Join(home, ".hydra")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	evilTarget := filepath.Join(home, "evil-target")
	if err := os.WriteFile(evilTarget, []byte("do not touch"), 0600); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(dir, "token")
	if err := os.Symlink(evilTarget, tokenPath); err != nil {
		t.Fatal(err)
	}

	if _, err := saveToken("fresh-token"); err == nil {
		t.Fatal("expected saveToken to refuse a symlinked token path")
	}

	got, err := os.ReadFile(evilTarget)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "do not touch" {
		t.Errorf("symlink target was modified: %q", got)
	}
}

func TestSaveToken_RefusesSymlinkedDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows by default")
	}
	home := withTempHome(t)
	realDir := filepath.Join(home, "real-hydra-dir")
	if err := os.MkdirAll(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".hydra")
	if err := os.Symlink(realDir, dir); err != nil {
		t.Fatal(err)
	}

	if _, err := saveToken("fresh-token"); err == nil {
		t.Fatal("expected saveToken to refuse a symlinked .hydra directory")
	}
}

func TestAtomicWriteFile_LeavesOldTokenOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "token")
	if err := os.WriteFile(target, []byte("old-token\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// Pre-occupy the exact temp path this process/pid would use, so the
	// O_EXCL create inside atomicWriteFile deterministically fails before
	// it ever touches the real target.
	tmpPath := filepath.Join(dir, fmt.Sprintf(".token.tmp.%d", os.Getpid()))
	if err := os.WriteFile(tmpPath, []byte("collision"), 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpPath)

	err := atomicWriteFile(dir, target, []byte("new-token\n"))
	if err == nil {
		t.Fatal("expected atomicWriteFile to fail when the temp path is already occupied")
	}

	got, rerr := os.ReadFile(target)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(got) != "old-token\n" {
		t.Errorf("target = %q, want the old token preserved untouched", got)
	}
}

func TestAtomicWriteFile_Succeeds(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "token")

	if err := atomicWriteFile(dir, target, []byte("hello\n")); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello\n" {
		t.Errorf("target content = %q, want %q", b, "hello\n")
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("target perm = %o, want 0600", perm)
	}

	// No leftover temp file.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "token" {
			t.Errorf("unexpected leftover file in dir: %s", e.Name())
		}
	}
}
