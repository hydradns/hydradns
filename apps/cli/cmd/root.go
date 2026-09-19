package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hydradns/hydra-cli/api"
	"github.com/spf13/cobra"
)

var (
	apiURL string
	token  string
	client *api.Client
)

var rootCmd = &cobra.Command{
	Use:   "hydra",
	Short: "HydraDNS CLI — manage your DNS firewall",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client = api.New(apiURL, token)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultURL := "http://localhost:8080"
	if env := os.Getenv("HYDRA_API_URL"); env != "" {
		defaultURL = env
	}

	defaultToken := ""
	if env := os.Getenv("HYDRA_TOKEN"); env != "" {
		defaultToken = env
	} else if home, err := os.UserHomeDir(); err == nil {
		if t, err := os.ReadFile(filepath.Join(home, ".hydra", "token")); err == nil {
			defaultToken = strings.TrimSpace(string(t))
		}
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api", defaultURL, "HydraDNS API URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", defaultToken, "API token")
}
