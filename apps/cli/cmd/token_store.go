package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// saveToken persists an API token the same way for every command that
// obtains one (login, setup): ~/.hydra/token, dir 0700, file 0600.
//
// ~/.hydra can pre-date this code (an older CLI version, a shared
// appliance account, or an attacker with local access), so this does not
// trust whatever it finds there:
//
//   - The directory is chmod'd to 0700 unconditionally, even if it
//     already existed with looser permissions (os.MkdirAll only sets the
//     mode when it *creates* the directory).
//   - A symlinked ~/.hydra directory, or a symlinked token path, is
//     refused outright rather than followed.
//   - Where the platform can portably tell us, a directory not owned by
//     the current user is refused too (skipped on Windows, which has no
//     equivalent to a POSIX UID via os.FileInfo).
//   - The token is written to a same-directory temp file created with
//     O_EXCL (so it can never be a followed symlink or a collision with
//     another writer), fsynced, then renamed over the target. A failure
//     anywhere in that sequence leaves the previous token file untouched
//     — there is no window where the target is truncated but not yet
//     rewritten.
func saveToken(tok string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".hydra")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", dir, err)
	}
	if err := secureDir(dir); err != nil {
		return "", err
	}

	tokenPath := filepath.Join(dir, "token")
	if err := refuseSymlink(tokenPath); err != nil {
		return "", err
	}

	if err := atomicWriteFile(dir, tokenPath, []byte(tok+"\n")); err != nil {
		return "", err
	}

	return tokenPath, nil
}

// secureDir refuses a symlinked or other-owned directory and
// unconditionally chmods it to 0700.
func secureDir(dir string) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", dir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use %s: it is a symlink", dir)
	}
	if err := checkOwnedByCurrentUser(dir, fi); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dir, err)
	}
	return nil
}

// refuseSymlink refuses to write through a symlinked path. A path that
// does not exist yet is fine — that's the normal first-write case.
func refuseSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write token: %s is a symlink", path)
	}
	return nil
}

// atomicWriteFile writes data to target by first writing to a temp file
// in dir (same filesystem as target, so the final rename is atomic and
// cannot cross a mount boundary), created with O_EXCL, fsynced, then
// renamed over target. The temp name is derived from the current PID
// rather than randomised: a same-PID collision can only happen if this
// process itself already has one in flight, which O_EXCL will correctly
// reject rather than silently reuse.
func atomicWriteFile(dir, target string, data []byte) (err error) {
	tmpPath := filepath.Join(dir, fmt.Sprintf(".token.tmp.%d", os.Getpid()))

	tmp, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temp token file: %w", err)
	}
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	if _, werr := tmp.Write(data); werr != nil {
		tmp.Close()
		err = fmt.Errorf("failed to write token: %w", werr)
		return err
	}
	if serr := tmp.Sync(); serr != nil {
		tmp.Close()
		err = fmt.Errorf("failed to sync token file: %w", serr)
		return err
	}
	if cerr := tmp.Close(); cerr != nil {
		err = fmt.Errorf("failed to close temp token file: %w", cerr)
		return err
	}
	if rerr := os.Rename(tmpPath, target); rerr != nil {
		err = fmt.Errorf("failed to save token: %w", rerr)
		return err
	}
	return nil
}
