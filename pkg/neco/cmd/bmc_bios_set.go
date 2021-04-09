package cmd

import (
	"github.com/spf13/cobra"
)

var bmcBiosSetCmd = &cobra.Command{
	Use:   "set",
	Short: "BMC bios",
	Long:  `BMC bios`,
}

func init() {
	bmcBiosCmd.AddCommand(bmcBiosSetCmd)
}
