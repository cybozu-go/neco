package cmd

import (
	"github.com/spf13/cobra"
)

var tpmCmd = &cobra.Command{
	Use:   "tpm",
	Short: "tpm related commands",
	Long:  `tpm`,
}

func init() {
	rootCmd.AddCommand(tpmCmd)
}
