package cmd

import (
	"github.com/spf13/cobra"
)

var bmcConfigSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "store a BMC configuration",
	Long: `Store a BMC configuration.

Possible keys are:
    bmc-user        - bmc-user.json contents.
    ipmi-user       - IPMI username for power management.
    ipmi-password   - IPMI password for power management.
    repair-user     - BMC username for repair operations.
    repair-password - BMC password for repair operations.
`,
}

func init() {
	bmcConfigCmd.AddCommand(bmcConfigSetCmd)
}
