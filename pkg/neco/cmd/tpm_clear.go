package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// Redfish API endpoints used for clearing TPM device on Dell equipment.
// These values will probably not be changed. So define as constants.
// If you want to get these values dynamically, you can get them as follows.
//
// $ curl --insecure -sS -X GET -u $BMC_USER:$BMC_PASS \
//        https://$BMC_ADDR/redfish/v1/Managers/iDRAC.Embedded.1 | jq .Links.Oem.Dell.Jobs
// {
//   "@odata.id": "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"
//  }

// $ curl --insecure -sS -X GET -u $BMC_USER:$BMC_PASS \
//        https://$BMC_ADDR/redfish/v1/Systems/System.Embedded.1/Bios | jq '."@Redfish.Settings".SettingsObject'
// {
//   "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Bios/Settings"
// }
const (
	jobURI          = "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"
	biosSettingsURI = "/redfish/v1/Systems/System.Embedded.1/Bios/Settings"
)

func setTpmAttribute(client *gofish.APIClient) error {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return err
	}
	bios, err := system.Bios()
	if err != nil {
		return err
	}

	attr := redfish.BiosAttributes{
		"Tpm2Hierarchy": "Clear",
	}
	return bios.UpdateBiosAttributes(attr)
}

func createBiosSettingsJob(client *gofish.APIClient) (string, error) {
	payload := map[string]string{
		"TargetSettingsURI": biosSettingsURI,
	}
	resp, err := client.Post(jobURI, payload)
	if err != nil {
		return "", err
	}
	jobURL, err := resp.Location()
	if err != nil {
		return "", err
	}
	split := strings.Split(jobURL.Path, "/")
	jobID := split[len(split)-1]
	return jobID, nil
}

func startOrRestart(client *gofish.APIClient) error {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return err
	}

	var resetType redfish.ResetType
	switch system.PowerState {
	case redfish.OnPowerState:
		resetType = redfish.ForceRestartResetType
	case redfish.OffPowerState:
		resetType = redfish.OnResetType
	default:
		// PoweringOnPowerState or PoweringOffPowerState
		return fmt.Errorf("unsupported power state: %s", system.PowerState)
	}

	return system.Reset(resetType)
}

func clearTpm(client *gofish.APIClient) error {
	err := setTpmAttribute(client)
	if err != nil {
		return err
	}
	log.Info("bios attribute is updated", nil)

	jobId, err := createBiosSettingsJob(client)
	if err != nil {
		return err
	}
	log.Info("bios setting job is created", map[string]interface{}{
		"job_id": jobId,
	})

	err = startOrRestart(client)
	if err != nil {
		return err
	}
	log.Info("machine power operation has been performed", nil)
	return nil
}

var tpmClearCmd = &cobra.Command{
	Use:   "clear SERIAL|IP",
	Short: "clear TPM devices on a machine",
	Long: `Clear TPM devices on a machine.
	
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

			return clearTpm(client)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	tpmCmd.AddCommand(tpmClearCmd)
}
