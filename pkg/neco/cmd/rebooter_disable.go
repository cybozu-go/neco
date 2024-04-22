package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rebooterDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable reboot list processing",
	Long:  `Disable reboot queue processing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return necoStorage.EnableNecoRebooter(ctx, false)
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterDisableCmd)

}
