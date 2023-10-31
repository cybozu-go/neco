package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var bmcRepairDellResetIdracCmd = &cobra.Command{
	Use:   "reset-idrac SERIAL_OR_IP",
	Short: "reset an iDRAC",
	Long:  `Reset the iDRAC of a machine having "SERIAL" or "IP" address.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		ctx := context.Background()
		bmc, err := getBMCWithType(ctx, args[0], "iDRAC")
		if err != nil {
			return err
		}

		client, err := dialToBMCByRepairUser(ctx, bmc)
		if err != nil {
			return err
		}
		defer client.Close()

		_, err = sshSessionOutput(client, "racadm racreset soft")
		return err
	},
}

func init() {
	bmcRepairDellCmd.AddCommand(bmcRepairDellResetIdracCmd)
}
