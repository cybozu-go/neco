package cmd

import (
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "store a configuration value to etcd",
	Long: `Store a configuration value to etcd.

Possible keys are:
    env                   - "staging" or "prod".  Default is "staging".
    slack                 - Slack WebHook URL.
    proxy                 - HTTP proxy server URL to access Internet.
    check-update-interval - Polling interval for checking new neco release.
    worker-timeout        - Timeout value to wait for workers.`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
