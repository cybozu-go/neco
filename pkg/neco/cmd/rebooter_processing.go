package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rebooterShowProcessingGroupCmd = &cobra.Command{
	Use:   "show-processing-group",
	Short: "show the processing group",
	Long:  `Show the processing group.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		pg, err := necoStorage.GetProcessingGroup(ctx)
		if err != nil {
			return err
		}
		fmt.Println(pg)
		return nil
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterShowProcessingGroupCmd)

}
