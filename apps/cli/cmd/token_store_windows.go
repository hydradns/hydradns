//go:build windows

package cmd

import "os"

// checkOwnedByCurrentUser is a no-op on Windows: os.FileInfo exposes no
// portable POSIX-style owner UID to compare against the current user, and
// a user's home directory is already ACL-restricted to that user (and
// Administrators) by default without our help.
func checkOwnedByCurrentUser(path string, fi os.FileInfo) error {
	return nil
}
