package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

func getBiosAttributes(client *gofish.APIClient) (redfish.BiosAttributes, error) {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return nil, err
	}
	bios, err := system.Bios()
	if err != nil {
		return nil, err
	}
	return bios.Attributes, nil
}

var bmcBiosGetAttributesCmd = &cobra.Command{
	Use:   "attributes SERIAL|IP [ATTRIBUTES_NAME...]",
	Short: "get bios attributes",
	Long:  `get bios attributes`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			client, err := getRedfishClient(ctx, args[0])
			if err != nil {
				return err
			}
			defer client.Logout()

			allAttrs, err := getBiosAttributes(client)
			if err != nil {
				return err
			}

			attrNames := args[1:]
			if len(attrNames) == 0 {
				e := json.NewEncoder(cmd.OutOrStdout())
				e.SetIndent("", "  ")
				return e.Encode(allAttrs)
			}

			showAttrs := redfish.BiosAttributes{}
			for _, name := range attrNames {
				val, ok := allAttrs[name]
				if !ok {
					return fmt.Errorf("invalid attribute: %s", name)
				}
				showAttrs[name] = val
			}

			e := json.NewEncoder(cmd.OutOrStdout())
			e.SetIndent("", "  ")
			return e.Encode(showAttrs)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcBiosGetCmd.AddCommand(bmcBiosGetAttributesCmd)
}
