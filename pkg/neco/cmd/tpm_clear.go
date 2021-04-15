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

const machineTypeLabelName = "machine-type"

const (
	tpmClearLogicTypeNotImplemented = iota
	tpmClearLogicTypeNothing
	tpmClearLogicTypeDellRedfish
)

var supportedMachineTypes = map[string]int{
	"qemu":        tpmClearLogicTypeNothing,     // Placemat VM. Clear logic is not implemented on placemat.
	"r640-boot-1": tpmClearLogicTypeNothing,     // Dell, TPM 1.2
	"r640-boot-2": tpmClearLogicTypeDellRedfish, // Dell, TPM 2.0
	"r640-cs-1":   tpmClearLogicTypeNothing,     // Dell, TPM 1.2
	"r640-cs-2":   tpmClearLogicTypeDellRedfish, // Dell, TPM 2.0
	"r740xd-ss-1": tpmClearLogicTypeNothing,     // Dell, TPM 1.2
	"r740xd-ss-2": tpmClearLogicTypeDellRedfish, // Dell, TPM 2.0
}

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
	dellRedfishJobURI          = "/redfish/v1/Managers/iDRAC.Embedded.1/Jobs"
	dellRedfishBiosSettingsURI = "/redfish/v1/Systems/System.Embedded.1/Bios/Settings"
)

func dellRedfishSetTpmAttribute(client *gofish.APIClient) error {
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

func dellRedfishCreateBiosSettingsJob(client *gofish.APIClient) (string, error) {
	payload := map[string]string{
		"TargetSettingsURI": dellRedfishBiosSettingsURI,
	}
	resp, err := client.Post(dellRedfishJobURI, payload)
	if err != nil && !strings.Contains(err.Error(), "SYS011") {
		// When a job has already been registered, "SYS011" will be returned.
		// We face this error when we re-execute "neco tpm clear" command due to the failure of the machine restart.
		// In order to succeed the re-executed command, ignore the "SYS011" error.
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

func dellRedfishClearTpm(ctx context.Context, bmcAddr string) error {
	client, err := getRedfishClient(ctx, bmcAddr)
	if err != nil {
		return fmt.Errorf("failed to get redfish client: %s", err.Error())
	}
	defer client.Logout()

	err = dellRedfishSetTpmAttribute(client)
	if err != nil {
		return fmt.Errorf("failed to set bios attribute: %s", err.Error())
	}
	log.Info("bios attribute is updated", nil)

	jobId, err := dellRedfishCreateBiosSettingsJob(client)
	if err != nil {
		return fmt.Errorf("failed to create bios setting job: %s", err.Error())
	}
	log.Info("bios setting job is created", map[string]interface{}{
		"job_id": jobId,
	})

	err = startOrRestart(client)
	if err != nil {
		return fmt.Errorf("failed to reset: %s", err.Error())
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

			machineType := machine.Spec.Labels[machineTypeLabelName]
			logicType, ok := supportedMachineTypes[machineType]
			if !ok {
				return fmt.Errorf("unknown machine type: machine-type=%s", machineType)
			}

			switch logicType {
			case tpmClearLogicTypeNotImplemented:
				return fmt.Errorf("clear logic is not implemented: machine-type=%s", machineType)
			case tpmClearLogicTypeNothing:
				log.Info("nothing to do", nil)
				return nil
			case tpmClearLogicTypeDellRedfish:
				return dellRedfishClearTpm(ctx, machine.Spec.BMC.IPv4)
			default:
				return fmt.Errorf("unknown logic type: %d", logicType)
			}
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
