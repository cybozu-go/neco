package cmd

import (
	"github.com/spf13/cobra"
)

// configGetCmd implements "neco config get"
var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "show the current configuration value",
	Long: `Show the current configuration value.

Possible keys are:
    env                   - "staging" or "prod".  Default is "staging".
    slack                 - Slack WebHook URL.
    proxy                 - HTTP proxy server URL to access Internet.
    check-update-interval - Polling interval for checking new neco release.
    worker-timeout        - Timeout value to wait for workers.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
}
