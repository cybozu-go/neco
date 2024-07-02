package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var leaderCmd = &cobra.Command{
	Use:   "leader",
	Short: "show the hostname of the current leader process",
	Long:  `Show the hostname of the current leader process.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			leader, err := necoStorage.GetNecoRebooterLeader(ctx)
			if err != nil {
				return err
			}

			fmt.Println(leader)
			return nil
		})
		well.Stop()
		return well.Wait()
	},
}

func init() {
	rebooterCmd.AddCommand(leaderCmd)
}
