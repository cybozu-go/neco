package cmd

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
)

var rebooterCancelCmd = &cobra.Command{
	Use:   "cancel INDEX",
	Short: "cancel the specified reboot list entry",
	Long:  `Cancel the specified reboot list entry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		index, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		entry, err := necoStorage.GetRebootListEntry(ctx, index)
		if err != nil {
			return err
		}
		entry.Status = neco.RebootListEntryStatusCancelled
		return necoStorage.UpdateRebootListEntry(ctx, entry)
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterCancelCmd)

}
