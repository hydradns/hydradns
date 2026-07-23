package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hydradns/hydra-cli/selfupdate"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateURL       string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install the latest hydra release",
	Long: `Check the release feed for a newer hydra version and, if one exists,
download it, verify it against the published SHA-256 checksums, and atomically
replace the current binary. The previous binary is kept at "<binary>.old" for
rollback.

The feed defaults to the hydra-cli GitHub releases API and can be overridden
with the HYDRA_UPDATE_URL environment variable or the --url flag.

This runs only when invoked explicitly; hydra never auto-updates on other
commands. Use --check to report whether an update is available without applying
it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		feedURL := updateURL
		if feedURL == "" {
			feedURL = os.Getenv("HYDRA_UPDATE_URL")
		}

		up := selfupdate.New(selfupdate.Config{
			CurrentVersion: Version,
			FeedURL:        feedURL,
		})

		rel, err := up.LatestRelease(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		newer, err := selfupdate.IsNewer(Version, rel.Version())
		if err != nil {
			return fmt.Errorf("cannot compare versions: %w", err)
		}

		if !newer {
			fmt.Printf("hydra %s is up to date (latest: %s)\n", Version, rel.TagName)
			return nil
		}

		fmt.Printf("Update available: %s -> %s\n", Version, rel.TagName)

		if updateCheckOnly {
			fmt.Println("Run 'hydra update' to install. (--check: no changes made)")
			return nil
		}

		fmt.Println("Downloading and verifying checksum...")
		bin, err := up.DownloadVerified(cmd.Context(), rel)
		if err != nil {
			return fmt.Errorf("update aborted: %w", err)
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot locate current binary: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}

		backup, err := selfupdate.Apply(bin, exe)
		if err != nil {
			return fmt.Errorf("failed to apply update: %w", err)
		}

		fmt.Printf("Updated to %s.\n", rel.TagName)
		fmt.Printf("Previous binary saved at %s (rollback: mv %s %s)\n", backup, backup, exe)
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Only report whether an update is available; do not modify anything")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "Release feed URL (overrides HYDRA_UPDATE_URL and the default)")
	rootCmd.AddCommand(updateCmd)
}
