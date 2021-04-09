package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var bmcJobListCmd = &cobra.Command{
	Use:   "list SERIAL|IP",
	Short: "list idrac job of a machine",
	Long: `Control power of a machine using Redfish API.

	SERIAL is the serial number of the machine.
	IP is one of the IP addresses owned by the machine.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			client, err := getRedfishClient(ctx, args[0])
			if err != nil {
				return err
			}
			defer client.Logout()

			ids, err := getJobIDs(client)
			if err != nil {
				return nil
			}
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcJobCmd.AddCommand(bmcJobListCmd)
}
