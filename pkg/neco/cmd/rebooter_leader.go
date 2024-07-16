package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rebooterLeaderCmd = &cobra.Command{
	Use:   "leader",
	Short: "show the hostname of the current leader process",
	Long:  `Show the hostname of the current leader process.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		leader, err := necoStorage.GetNecoRebooterLeader(ctx)
		if err != nil {
			return err
		}
		fmt.Println(leader)
		return nil
	},
}

func init() {
	rebooterCmd.AddCommand(rebooterLeaderCmd)
}
