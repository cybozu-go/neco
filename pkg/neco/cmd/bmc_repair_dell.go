package cmd

import (
	"github.com/spf13/cobra"
)

var bmcRepairDellCmd = &cobra.Command{
	Use:   "dell",
	Short: "repair a Dell machine via BMC",
	Long:  `Try to repair a Dell machine by invoking BMC functions remotely.`,
}

func init() {
	bmcRepairCmd.AddCommand(bmcRepairDellCmd)
}
