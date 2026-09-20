package cmd

import (
	"bufio"
	"fmt"
	"os"
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

		// Save token to ~/.hydra/token (same helper `hydra setup` uses).
		tokenPath, err := saveToken(resp.Token)
		if err != nil {
			return err
		}

		fmt.Printf("Logged in successfully. Token saved to %s\n", tokenPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
