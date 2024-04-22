package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
)

var rebooterCancelAllCmd = &cobra.Command{
	Use:   "cancel-all",
	Short: "cancel all the reboot list entries",
	Long:  `Cancel all the reboot list entries.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		entries, err := necoStorage.GetRebootListEntries(ctx)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entry.Status = neco.RebootListEntryStatusCancelled
			err := necoStorage.UpdateRebootListEntry(ctx, entry)
			if err != nil {
				fmt.Printf("failed to cancel the enty. index:%d, node:%s\n", entry.Index, entry.Node)
				return err
			}
		}
		return nil
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterCancelAllCmd)

}
