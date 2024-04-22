package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rebooterEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "enable reboot list processing",
	Long:  `Enable reboot list processing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return necoStorage.EnableNecoRebooter(ctx, true)
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterEnableCmd)

}
