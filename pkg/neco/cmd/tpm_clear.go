package cmd

import (
	"context"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	jobUri          = "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"
	biosSettingsUri = "/redfish/v1/Systems/System.Embedded.1/Bios/Settings"
)

/*
$ curl --insecure -sS -X GET -u $BMC_USER:$BMC_PASS \
       https://$BMC_ADDR/redfish/v1/Managers/iDRAC.Embedded.1 | jq .Links.Oem.Dell.Jobs
{
  "@odata.id": "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"
 }

$ curl --insecure -sS -X GET -u $BMC_USER:$BMC_PASS \
       https://$BMC_ADDR/redfish/v1/Systems/System.Embedded.1/Bios | jq '."@Redfish.Settings".SettingsObject'
{
  "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Bios/Settings"
}
*/

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
		"TargetSettingsURI": biosSettingsUri,
	}
	resp, err := client.Post(jobUri, payload)
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

func restart(client *gofish.APIClient) error {
	system, err := getComputerSystem(client.Service)
	if err != nil {
		return err
	}
	return system.Reset(redfish.ForceRestartResetType)
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
	log.Info("bios setting job is cteated", map[string]interface{}{
		"job_id": jobId,
	})

	err = restart(client)
	if err != nil {
		return err
	}
	log.Info("machine is restared", nil)
	return nil
}

var tpmClearCmd = &cobra.Command{
	Use:   "clear SERIAL|IP",
	Short: "clear",
	Long:  `clear`,

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
