package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// saveToken persists an API token the same way for every command that
// obtains one (login, setup): ~/.hydra/token, dir 0700, file 0600. Kept
// in one place so a permissions fix (or a future token-store change)
// only has to happen once.
func saveToken(tok string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".hydra")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", dir, err)
	}

	tokenPath := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenPath, []byte(tok+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed to write token: %w", err)
	}

	return tokenPath, nil
}
