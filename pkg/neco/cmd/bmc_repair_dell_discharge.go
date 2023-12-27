package cmd

import (
	"context"
	"errors"
	"regexp"

	"github.com/spf13/cobra"
)

var bmcRepairDellDischargeCmd = &cobra.Command{
	Use:   "discharge SERIAL_OR_IP",
	Short: "discharge a machine",
	Long: `Simulate power-disconnection and discharge of a machine having "SERIAL" or "IP" address.
This implies reboot of the machine.`,

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

		output, err := sshSessionOutput(client, "racadm set BIOS.MiscSettings.PowerCycleRequest FullPowerCycle")
		if err != nil {
			return err
		}

		re := regexp.MustCompile(`(?s).*\[Key=(.*)#MiscSettings\].*`)
		if !re.Match(output) {
			return errors.New("FQDD of BIOS setup not found")
		}
		fqdd := re.ReplaceAllString(string(output), "$1")

		_, err = sshSessionOutput(client, "racadm jobqueue create "+fqdd)
		if err != nil {
			return err
		}

		_, err = sshSessionOutput(client, "racadm serveraction powercycle")
		return err
	},
}

func init() {
	bmcRepairDellCmd.AddCommand(bmcRepairDellDischargeCmd)
}
