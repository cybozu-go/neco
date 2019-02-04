package cmd

import (
	"github.com/spf13/cobra"
)

var bmcConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "BMC configurations",
	Long:  `BMC configurations`,
}

func init() {
	bmcCmd.AddCommand(bmcConfigCmd)
}
