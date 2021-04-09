package cmd

import (
	"context"
	"encoding/json"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

func setBiosAttributes(client *gofish.APIClient, attrs redfish.BiosAttributes) error {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return err
	}
	bios, err := system.Bios()
	if err != nil {
		return err
	}
	return bios.UpdateBiosAttributes(attrs)
}

var bmcBiosSetAttributesCmd = &cobra.Command{
	Use:   "attributes SERIAL|IP ATTRIBUTES",
	Short: "set bios attributes",
	Long:  `set bios attributes`,

	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			var attrs redfish.BiosAttributes
			err := json.Unmarshal([]byte(args[1]), &attrs)
			if err != nil {
				return err
			}

			client, err := getRedfishClient(ctx, args[0])
			if err != nil {
				return err
			}
			defer client.Logout()

			return setBiosAttributes(client, attrs)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcBiosSetCmd.AddCommand(bmcBiosSetAttributesCmd)
}
