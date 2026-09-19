package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "Show or control the DNS engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := client.GetEngineStatus()
		if err != nil {
			return err
		}
		status := "STOPPED"
		if s.AcceptingQueries {
			status = "RUNNING"
		}
		fmt.Printf("Engine: %s (enabled=%v)\n", status, s.Enabled)
		if s.LastError != "" {
			fmt.Printf("Error:  %s\n", s.LastError)
		}
		return nil
	},
}

var engineEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable the DNS engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.ToggleEngine(true); err != nil {
			return err
		}
		fmt.Println("DNS engine enabled")
		return nil
	},
}

var engineDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable the DNS engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.ToggleEngine(false); err != nil {
			return err
		}
		fmt.Println("DNS engine disabled")
		return nil
	},
}

func init() {
	engineCmd.AddCommand(engineEnableCmd)
	engineCmd.AddCommand(engineDisableCmd)
	rootCmd.AddCommand(engineCmd)
}
