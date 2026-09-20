//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

// checkOwnedByCurrentUser refuses a directory not owned by the current
// user. Only meaningful on POSIX systems that expose ownership via
// syscall.Stat_t; see token_store_windows.go for the no-op counterpart.
func checkOwnedByCurrentUser(path string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		// Platform doesn't expose ownership this way; nothing to check.
		return nil
	}
	if int(st.Uid) != os.Getuid() {
		return fmt.Errorf("refusing to use %s: owned by uid %d, not the current user (uid %d)", path, st.Uid, os.Getuid())
	}
	return nil
}
