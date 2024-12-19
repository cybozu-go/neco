package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/spf13/cobra"
)

type fwNameConversionRule struct {
	re      *regexp.Regexp
	newName string
}

var fwNameConversionRuleList = map[string][]fwNameConversionRule{
	"r6525-boot-1": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell EMC iDRAC Service Module Embedded Package .*`), "Dell EMC iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile(`Power Supply.Slot.[12]`), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .*`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Broadcom NetXtreme Gigabit Ethernet"},
		// Disk
		{regexp.MustCompile(`PCIe SSD in Slot [0-9] in Bay 1`), "PCIe SSD"},
	},
	"r6525-cs-1": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Broadcom NetXtreme Gigabit Ethernet"},
		// Disk
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r6525-cs-2": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Broadcom NetXtreme Gigabit Ethernet"},
		// Disk
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r7525-ss-1": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Broadcom NetXtreme Gigabit Ethernet"},
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		// Disk
		{regexp.MustCompile(`Disk [0-9] on AHCI Controller in SL 7`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r7525-ss-2": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Broadcom NetXtreme Gigabit Ethernet"},
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		// Disk
		{regexp.MustCompile(`Disk [0-9] on AHCI Controller in SL 7`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r6615-cs-1": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet \(BCM5720\) - .*`), "Broadcom NetXtreme Gigabit Ethernet (BCM5720)"},
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		// Disk
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r7615-ss-1": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet \(BCM5720\) - .*`), "Broadcom NetXtreme Gigabit Ethernet (BCM5720)"},
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		// Disk
		{regexp.MustCompile(`Disk [0-9] on BOSS in SL 10`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r7615-ss-2": {
		// Dell
		{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
		{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
		{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
		// Power
		{regexp.MustCompile("Power Supply.Slot.[12]"), "Power Supply"},
		// NIC
		{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet \(BCM5720\) - .*`), "Broadcom NetXtreme Gigabit Ethernet (BCM5720)"},
		{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Intel(R) Ethernet 25G 2P E810-XXV OCP"},
		// Disk
		{regexp.MustCompile(`Disk [0-9] on BOSS in SL 10`), "BOSS"},
		{regexp.MustCompile("PCIe SSD in Slot [0-9][0-9]* in Bay 1"), "PCIe SSD"},
	},
}

type firmwareInfo struct {
	name    string
	version string
}

var collectFirmwareGetOpts sabakanMachinesGetOpts

var collectFirmwareCmd = &cobra.Command{
	Use:   "firmware SERIAL|IP...",
	Short: "collect Firmware versions on a machine",
	Long: `collect Firmware versions on a machine.

SERIAL is the serial number of the machine.
IP is one of the IP addresses owned by the machine.`,

	Run: func(cmd *cobra.Command, args []string) {
		bmcUser := os.Getenv("BMC_USER")
		if bmcUser == "" {
			log.ErrorExit(errors.New("BMC_USER is not set"))
		}
		bmcPassword := os.Getenv("BMC_PASS")
		if bmcPassword == "" {
			log.ErrorExit(errors.New("BMC_PASS is not set"))
		}

		machines, err := sabakanMachinesGet(cmd.Context(), &collectFirmwareGetOpts)
		if err != nil {
			log.ErrorExit(err)
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		fwVersions := map[string][]string{} // key: device, value: versions
		for _, m := range machines {
			machineType := m.Spec.Labels["machine-type"]
			rules := fwNameConversionRuleList[machineType]

			fwList, err := collectFirmware(cmd.Context(), httpClient, m.Spec.BMC.IPv4, bmcUser, bmcPassword)
			if err != nil {
				log.ErrorExit(err)
			}

			for _, fw := range fwList {
				name := fw.name
				for _, r := range rules {
					if r.re.MatchString(fw.name) {
						name = r.newName
						break
					}
				}
				newVersions := append(fwVersions[name], fw.version)
				slices.Sort(newVersions)
				fwVersions[name] = slices.Compact(newVersions)
			}
		}

		fwNames := []string{}
		for k := range fwVersions {
			fwNames = append(fwNames, k)
		}
		slices.Sort(fwNames)

		for _, name := range fwNames {
			fmt.Printf("%s: %s\n", name, strings.Join(fwVersions[name], ", "))
		}
	},
}

func collectFirmware(ctx context.Context, httpClient *http.Client, bmcAddr, bmcUser, bmcPassword string) ([]firmwareInfo, error) {
	request := func(path string) ([]byte, error) {
		request, err := http.NewRequest("GET", "https://"+bmcAddr+path, nil)
		if err != nil {
			return nil, err
		}
		request.SetBasicAuth(bmcUser, bmcPassword)

		response, err := httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		return io.ReadAll(response.Body)
	}

	data, err := request("/redfish/v1/UpdateService/FirmwareInventory")
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware inventory collection: %w", err)
	}

	fwCollection := struct {
		Members []struct {
			Id string `json:"@odata.id"`
		} `json:"Members"`
	}{}
	err = json.Unmarshal([]byte(data), &fwCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal firmware inventory collection: %w", err)
	}

	installedFirmwarePathes := []string{}
	for _, ent := range fwCollection.Members {
		if strings.Contains(ent.Id, "/Installed-") {
			installedFirmwarePathes = append(installedFirmwarePathes, ent.Id)
		}
	}

	ret := []firmwareInfo{}
	for _, fwPath := range installedFirmwarePathes {
		data, err := request(fwPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get firmware inventory: %w", err)
		}

		fwInfo := struct {
			Name    string `json:"Name"`
			Version string `json:"Version"`
		}{}
		err = json.Unmarshal([]byte(data), &fwInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal firmware inventor: %w", err)
		}
		ret = append(ret, firmwareInfo{name: fwInfo.Name, version: fwInfo.Version})
	}
	return ret, nil
}

func init() {
	collectCmd.AddCommand(collectFirmwareCmd)
	addSabakanMachinesGetOpts(collectFirmwareCmd, &collectFirmwareGetOpts)
}
