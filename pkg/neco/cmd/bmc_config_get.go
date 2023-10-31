package cmd

import (
	"github.com/spf13/cobra"
)

var bmcConfigGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "show the current BMC configuration value",
	Long: `Show the current BMC configuration value.

Possible keys are:
    bmc-user        - bmc-user.json contents.
    ipmi-user       - IPMI username for power management.
    ipmi-password   - IPMI password for power management.
    repair-user     - BMC username for repair operations.
    repair-password - BMC password for repair operations.
`,
}

func init() {
	bmcConfigCmd.AddCommand(bmcConfigGetCmd)
}
