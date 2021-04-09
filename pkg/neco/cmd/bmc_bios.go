package cmd

import (
	"github.com/spf13/cobra"
)

var bmcBiosCmd = &cobra.Command{
	Use:   "bios",
	Short: "BMC bios",
	Long:  `BMC bios`,
}

func init() {
	bmcCmd.AddCommand(bmcBiosCmd)
}
