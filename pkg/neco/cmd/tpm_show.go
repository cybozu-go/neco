package cmd

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

type tpmSettings struct {
	Devices    []redfish.TrustedModules
	Attributes redfish.BiosAttributes
}

func getTpmSettings(client *gofish.APIClient) (*tpmSettings, error) {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return nil, err
	}

	// Get BIOS attributes related to TPM.
	tpmAttrs := redfish.BiosAttributes{}
	bios, err := system.Bios()
	if err != nil && !strings.Contains(err.Error(), "404") {
		// Ignore error if return 404. It means bios endpoint is not implemented.
		return nil, err
	}
	// When 404 error is returned, bios will be nil.
	// And in some cases, bios will be nil even if ComputerSystem.BIOS() doesn't return an error.
	// https://github.com/stmcginnis/gofish/blob/v0.7.0/redfish/computersystem.go#L635
	if bios != nil {
		for name, val := range bios.Attributes {
			if !strings.HasPrefix(name, "Tpm") {
				continue
			}
			tpmAttrs[name] = val
		}
	}

	return &tpmSettings{
		Devices:    system.TrustedModules,
		Attributes: tpmAttrs,
	}, nil
}

var tpmShowCmd = &cobra.Command{
	Use:   "show SERIAL|IP",
	Short: "show TPM devices on a machine",
	Long: `Show TPM devices on a machine.

SERIAL is the serial number of the machine.
IP is one of the IP addresses owned by the machine.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			machine, err := lookupMachine(ctx, args[0])
			if err != nil {
				return err
			}

			client, err := getRedfishClient(ctx, machine.Spec.BMC.IPv4)
			if err != nil {
				return err
			}
			defer client.Logout()

			tpm, err := getTpmSettings(client)
			if err != nil {
				return err
			}

			e := json.NewEncoder(cmd.OutOrStdout())
			e.SetIndent("", "  ")
			return e.Encode(tpm)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	tpmCmd.AddCommand(tpmShowCmd)
}
