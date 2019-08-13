package cmd

import (
	"github.com/spf13/cobra"
)

var ckeWeightCmd = &cobra.Command{
	Use:   "weight",
	Short: "Role weight configurations",
	Long:  `Role weight configurations`,
}

func init() {
	ckeCmd.AddCommand(ckeWeightCmd)
}
