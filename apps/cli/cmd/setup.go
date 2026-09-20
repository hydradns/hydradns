package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/hydradns/hydra-cli/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// minSetupPasswordLength mirrors the server's validation
// (apps/core cmd/controlplane/handlers/auth.go: setupRequest.Password
// has `binding:"required,min=8"`). Enforced here too so a weak password
// never leaves the machine.
const minSetupPasswordLength = 8

var (
	setupEmail         string
	setupPasswordStdin bool
	setupShowToken     bool
)

// Seams so tests can drive the interactive/non-interactive branches
// without a real TTY. Mirrors the timeNow-style seam already used in
// setup_router.go.
var (
	setupStdin        io.Reader = os.Stdin
	setupIsTerminal             = func() bool { return term.IsTerminal(int(syscall.Stdin)) }
	setupReadPassword           = func() ([]byte, error) { return term.ReadPassword(int(syscall.Stdin)) }
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "First-boot setup: create the admin account and API token",
	Long: `Runs the one-time first-boot setup against a fresh HydraDNS instance:
creates the admin account and mints the API token every other hydra
command (and the dashboard) needs. This replaces hand-rolling a
curl + auth/setup + token-scraping pipeline.

This is NOT "hydra setup-router", which generates firewall rules for a
customer's router and assumes an admin already exists. Run "hydra setup"
once on a fresh install, "hydra login" for later sessions on any
machine, and "hydra setup-router" only once the box is already running.

There is deliberately no --password flag: a password on the command
line ends up in shell history and in "ps" output for every other user
on the box. Use the interactive prompt (asked twice, not echoed), or
pipe one line to stdin for scripts/CI:

  echo "correct horse battery staple" | hydra setup --password-stdin

Blocklist selection during setup is dashboard-only for now; this
command only creates the admin account and token.`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().StringVar(&setupEmail, "email", "", "Admin email (optional; server defaults to admin@hydradns.local)")
	setupCmd.Flags().BoolVar(&setupPasswordStdin, "password-stdin", false, "Read the password as a single line from stdin (for scripts/CI)")
	setupCmd.Flags().BoolVar(&setupShowToken, "show-token", false, "Print the API token to stdout (hidden by default)")
	setupCmd.SilenceUsage = true // runtime/auth errors here aren't "how do I use this" errors
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	status, err := client.GetAuthStatus()
	if err != nil {
		return setupFriendlyErr(err)
	}
	if status.SetupComplete {
		return fmt.Errorf("setup has already been completed on this instance — run `hydra login` instead (no password was sent)")
	}

	password, err := readSetupPassword()
	if err != nil {
		return err
	}
	if len(password) < minSetupPasswordLength {
		return fmt.Errorf("password must be at least %d characters (server requirement — nothing was sent)", minSetupPasswordLength)
	}

	resp, err := client.Setup(api.SetupRequest{
		Email:    setupEmail,
		Password: password,
	})
	if err != nil {
		return setupFriendlyErr(err)
	}

	tokenPath, err := saveToken(resp.Token)
	if err != nil {
		return err
	}

	fmt.Printf("Admin account created. Token saved to %s\n", tokenPath)
	for _, w := range resp.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	if setupShowToken {
		fmt.Printf("Token: %s\n", resp.Token)
	}
	fmt.Println()
	fmt.Println("Next: try `hydra block <domain>` or open the dashboard to finish configuring blocklists.")
	return nil
}

// readSetupPassword resolves the password from --password-stdin, or an
// interactive double-prompt when stdin is a terminal. There is no flag
// for a plaintext password (see setupCmd.Long).
func readSetupPassword() (string, error) {
	if setupPasswordStdin {
		scanner := bufio.NewScanner(setupStdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("failed to read password from stdin: %w", err)
			}
			return "", fmt.Errorf("no password received on stdin")
		}
		return strings.TrimSpace(scanner.Text()), nil
	}

	if !setupIsTerminal() {
		return "", fmt.Errorf("stdin is not a terminal; pipe a password with --password-stdin (there is no --password flag: it would leak into shell history and process listings)")
	}

	fmt.Print("Choose an admin password: ")
	raw1, err := setupReadPassword()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	fmt.Print("Confirm password: ")
	raw2, err := setupReadPassword()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	p1 := strings.TrimSpace(string(raw1))
	p2 := strings.TrimSpace(string(raw2))
	if p1 != p2 {
		return "", fmt.Errorf("passwords do not match")
	}
	return p1, nil
}

// setupFriendlyErr rewrites the client's generic error strings into
// something actionable for a first-boot user staring at a broken demo.
func setupFriendlyErr(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection failed"):
		return fmt.Errorf("cannot reach the HydraDNS API at %s: %w\n  check: `docker compose ps`, and that --api / HYDRA_API_URL points at the controlplane", apiURL, err)
	case strings.Contains(msg, "invalid response"):
		return fmt.Errorf("got a non-JSON response from %s: %w\n  is --api / HYDRA_API_URL pointing at the HydraDNS API (not the dashboard)?", apiURL, err)
	case strings.Contains(msg, "setup already completed"):
		return fmt.Errorf("setup was completed by another process just now — run `hydra login` instead")
	default:
		return err
	}
}
