package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rebooterIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled",
	Short: "show reboot list status",
	Long:  `Show whether the processing of the reboot list is enabled or not.  "true" if enabled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		enabled, err := necoStorage.IsNecoRebooterEnabled(ctx)
		if err != nil {
			return err
		}
		fmt.Println(enabled)
		return nil
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterIsEnabledCmd)
}
