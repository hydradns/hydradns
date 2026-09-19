package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the HydraDNS API",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("Password: ")

		var password string
		if term.IsTerminal(int(syscall.Stdin)) {
			raw, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(raw)
			fmt.Println() // newline after hidden input
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				password = scanner.Text()
			}
		}

		password = strings.TrimSpace(password)
		if password == "" {
			return fmt.Errorf("password cannot be empty")
		}

		// Call login endpoint
		resp, err := client.Login(password)
		if err != nil {
			return err
		}

		// Save token to ~/.hydra/token
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		dir := filepath.Join(home, ".hydra")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}

		tokenPath := filepath.Join(dir, "token")
		if err := os.WriteFile(tokenPath, []byte(resp.Token+"\n"), 0600); err != nil {
			return fmt.Errorf("failed to write token: %w", err)
		}

		fmt.Printf("Logged in successfully. Token saved to %s\n", tokenPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
