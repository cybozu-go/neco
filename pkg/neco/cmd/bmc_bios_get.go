package cmd

import (
	"github.com/spf13/cobra"
)

var bmcBiosGetCmd = &cobra.Command{
	Use:   "get",
	Short: "BMC bios",
	Long:  `BMC bios`,
}

func init() {
	bmcBiosCmd.AddCommand(bmcBiosGetCmd)
}
