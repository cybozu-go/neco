package cmd

import (
	"github.com/spf13/cobra"
)

var bmcJobCmd = &cobra.Command{
	Use:   "job",
	Short: "BMC job",
	Long:  `BMC job`,
}

func init() {
	bmcCmd.AddCommand(bmcJobCmd)
}
