package cmd

import (
	"github.com/spf13/cobra"
)

var bmcConfigSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "store a BMC configuration",
	Long: `Store a BMC configuration.

Possible keys are:
    bmc-user      - bmc-user.json contents.
    ipmi-user     - IPMI username for power management.
    ipmi-password - IPMI password for power management.
`,
}

func init() {
	bmcConfigCmd.AddCommand(bmcConfigSetCmd)
}
