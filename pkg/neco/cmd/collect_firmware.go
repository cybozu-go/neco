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
	"sync"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

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
		collector := newFirmwareCollector(bmcUser, bmcPassword)

		well.Go(func(ctx context.Context) error {
			machines, err := sabakanMachinesGet(ctx, &collectFirmwareGetOpts)
			if err != nil {
				log.ErrorExit(err)
			}

			/*
				 "machine-type-1":
				     "BIOS":
					     "1.1.1": 1
						 "2.2.2": 2
				     "Integrated NIC":
					     "1.2.3": 3
				 "machine-type-2":
				     "BIOS":
					     "1.1.1": 3
				     "Integrated NIC":
					     "1.2.3": 3
			*/
			summary := map[string]map[string]map[string]int{}

			var mu sync.Mutex
			eg, egctx := errgroup.WithContext(ctx)
			eg.SetLimit(10)
			for _, m := range machines {
				m := m
				eg.Go(func() error {
					fwInfoList, err := collector.fetchFirmwareInfo(egctx, &m)
					if err != nil {
						log.Error("failed to fetch firmware info", map[string]interface{}{"serial": m.Spec.Serial, "error": err.Error()})
						return nil // Don't stop other goroutines
					}

					mu.Lock()
					defer mu.Unlock()

					for _, info := range fwInfoList {
						fmt.Printf("%s,%s,%s,%s,%s\n", info.serial, info.machineType, info.alias, info.name, info.version)

						if _, ok := summary[info.machineType]; !ok {
							summary[info.machineType] = map[string]map[string]int{}
						}
						if _, ok := summary[info.machineType][info.alias]; !ok {
							summary[info.machineType][info.alias] = map[string]int{}
						}
						summary[info.machineType][info.alias][info.version]++
					}
					return nil
				})
			}
			if err := eg.Wait(); err != nil {
				log.ErrorExit(err)
			}

			fmt.Println("# Summary")
			machineTypeList := []string{}
			for machineType := range summary {
				machineTypeList = append(machineTypeList, machineType)
			}
			slices.Sort(machineTypeList)
			for _, machineType := range machineTypeList {
				fmt.Println("## " + machineType)
				aliasList := []string{}
				for alias := range summary[machineType] {
					aliasList = append(aliasList, alias)
				}
				slices.Sort(aliasList)
				for _, alias := range aliasList {
					s := []string{}
					for version, count := range summary[machineType][alias] {
						s = append(s, fmt.Sprintf("%s(%d)", version, count))
					}
					slices.Sort(s)
					fmt.Println(alias + ": " + strings.Join(s, ", "))
				}
				fmt.Println("")
			}
			return nil
		})
		well.Stop()
		if err := well.Wait(); err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	collectCmd.AddCommand(collectFirmwareCmd)
	addSabakanMachinesGetOpts(collectFirmwareCmd, &collectFirmwareGetOpts)
}

type fwNameAlias struct {
	re    *regexp.Regexp
	alias string
}

var fwNameCommonAliasList = []fwNameAlias{
	// Dell
	{regexp.MustCompile(`Dell 64 Bit uEFI Diagnostics, .*`), "Dell 64 Bit uEFI Diagnostics"},
	{regexp.MustCompile(`Dell OS Driver Pack, .*`), "Dell OS Driver Pack"},
	{regexp.MustCompile(`Dell EMC iDRAC Service Module Embedded Package .*`), "Dell EMC iDRAC Service Module Embedded Package"}, // old
	{regexp.MustCompile(`Dell iDRAC Service Module Embedded Package .*`), "Dell iDRAC Service Module Embedded Package"},
	// Power Supply
	{regexp.MustCompile(`Power Supply.Slot.[12]`), "Power Supply"},
	// NIC
	{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet - .*`), "Embedded NIC"},
	{regexp.MustCompile(`Broadcom NetXtreme Gigabit Ethernet \(BCM5720\) - .*`), "Embedded NIC"},
	{regexp.MustCompile(`Intel\(R\) Ethernet 25G 2P E810-XXV OCP - .+`), "Integrated NIC"},
}

var fwNameAliasList = map[string][]fwNameAlias{
	"r6525-boot-1": {
		{regexp.MustCompile(`PCIe SSD in Slot [0-9] in Bay 1`), "PCIe SSD"},
	},
	"r6525-cs-1": {
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r6525-cs-2": {
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r7525-ss-1": {
		{regexp.MustCompile(`Disk [0-9] on AHCI Controller in SL 7`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r7525-ss-2": {
		{regexp.MustCompile(`Disk [0-9] on AHCI Controller in SL 7`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r6615-cs-1": {
		{regexp.MustCompile("PCIe SSD in Slot [0-9] in Bay 1"), "PCIe SSD"},
	},
	"r7615-ss-1": {
		{regexp.MustCompile(`Disk [0-9] on BOSS in SL 10`), "BOSS"},
		{regexp.MustCompile(`Disk [0-9][0-9]* in Backplane 1 of Storage Controller in Slot 3`), "HDD"},
	},
	"r7615-ss-2": {
		{regexp.MustCompile(`Disk [0-9] on BOSS in SL 10`), "BOSS"},
		{regexp.MustCompile("PCIe SSD in Slot [0-9][0-9]* in Bay 1"), "PCIe SSD"},
	},
}

type firmwareCollector struct {
	client      *http.Client
	bmcUser     string
	bmcPassword string
}

type firmwareInfo struct {
	serial      string
	machineType string
	alias       string
	name        string
	version     string
}

func newFirmwareCollector(bmcUser, bmcPassword string) *firmwareCollector {
	return &firmwareCollector{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		bmcUser:     bmcUser,
		bmcPassword: bmcPassword,
	}
}

func (c *firmwareCollector) request(ctx context.Context, bmcAddr, path string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", "https://"+bmcAddr+path, nil) // TODO: set timeout
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(c.bmcUser, c.bmcPassword)

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}

func (c *firmwareCollector) fetchFirmwareInfo(ctx context.Context, machine *sabakan.Machine) ([]firmwareInfo, error) {
	data, err := c.request(ctx, machine.Spec.BMC.IPv4, "/redfish/v1/UpdateService/FirmwareInventory")
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware inventory collection: %w", err)
	}

	fwInventoryCollectionResp := struct {
		Members []struct {
			Id string `json:"@odata.id"`
		} `json:"Members"`
	}{}
	err = json.Unmarshal([]byte(data), &fwInventoryCollectionResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal firmware inventory collection: %w", err)
	}

	installedFirmwarePathes := []string{}
	for _, ent := range fwInventoryCollectionResp.Members {
		if strings.Contains(ent.Id, "/Installed-") {
			installedFirmwarePathes = append(installedFirmwarePathes, ent.Id)
		}
	}

	serial := machine.Spec.Serial
	machineType := machine.Spec.Labels["machine-type"]
	aliasList := append(fwNameCommonAliasList, fwNameAliasList[machineType]...)

	ret := []firmwareInfo{}
	for _, fwPath := range installedFirmwarePathes {
		data, err := c.request(ctx, machine.Spec.BMC.IPv4, fwPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get firmware inventory: %w", err)
		}

		fwInventoryResp := struct {
			Name    string `json:"Name"`
			Version string `json:"Version"`
		}{}
		err = json.Unmarshal([]byte(data), &fwInventoryResp)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal firmware inventory: %w", err)
		}

		alias := fwInventoryResp.Name
		for _, r := range aliasList {
			if r.re.MatchString(fwInventoryResp.Name) {
				alias = r.alias
				break
			}
		}
		ret = append(ret, firmwareInfo{
			serial:      serial,
			machineType: machineType,
			alias:       alias,
			name:        fwInventoryResp.Name,
			version:     fwInventoryResp.Version,
		})
	}
	return ret, nil
}
