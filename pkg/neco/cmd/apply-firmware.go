package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sort"
	"sync"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/cybozu-go/sabakan/v3/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-input"
	"github.com/vishvananda/netlink"
)

var applyFirmwareCmd = &cobra.Command{
	Use:   "apply-firmware UPDATER_FILE...",
	Short: "initiate firmware application",
	Long: `Send firmware updaters to BMCs and schedule reboot of workers.

This uses CKE's function of graceful reboot for the nodes used by CKE.
As for the other nodes, this reboots them immediately.
If some nodes are already powered off, this command does not do anything to those nodes.`,

	Args: cobra.MinimumNArgs(1),
	Run:  applyFirmwareRun,
}

var applyFirmwareGetOpts sabakanMachinesGetOpts
var applyFirmwareRebootOption bool

func applyFirmwareRun(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	uploadAssetsAndRunCommandOnWorkers(ctx, &applyFirmwareGetOpts, args, []string{"docker", "exec", "setup-hw", "setup-apply-firmware"}, applyFirmwareRebootOption)
}

func uploadAssetsAndRunCommandOnWorkers(ctx context.Context, getOpts *sabakanMachinesGetOpts, filenames []string, cmdline []string, needReboot bool) {
	machines, err := sabakanMachinesGet(ctx, getOpts)
	if err != nil {
		log.ErrorExit(err)
	}
	machinesWithoutBootServers := []sabakan.Machine{}
	for _, m := range machines {
		if m.Spec.Role != "boot" {
			machinesWithoutBootServers = append(machinesWithoutBootServers, m)
		}
	}
	machines = machinesWithoutBootServers
	fmt.Printf("Applying to %d machines.\n", len(machines))

	sabakanClient, err := client.NewClient(neco.SabakanLocalEndpoint, &http.Client{})
	if err != nil {
		log.ErrorExit(err)
	}

	hostAddr, err := getInterfaceAddress("node0")
	if err != nil {
		log.ErrorExit(err)
	}
	assetUrlRoot := "http://" + hostAddr.String() + ":10080/api/v1/assets/"
	assetUrls := []string{}
	for _, filename := range filenames {
		name := filepath.Base(filename)
		_, err := sabakanClient.AssetsUpload(ctx, name, filename, nil)
		if err != nil {
			log.ErrorExit(err)
		}
		assetUrl := assetUrlRoot + name
		log.Info("asset uploaded", map[string]interface{}{
			"file":      filename,
			"name":      name,
			"asset_url": assetUrl,
		})
		assetUrls = append(assetUrls, assetUrl)
	}

	var wg sync.WaitGroup
	var mtx sync.Mutex
	failedAddrs := []string{}
	succeededAddrs := []string{}
	succeededMachines := []sabakan.Machine{}
	for _, machine := range machines {
		wg.Add(1)
		go func(machine sabakan.Machine) {
			defer wg.Done()
			addr := machine.Spec.IPv4[0]
			cmdArgs := []string{"ssh", addr}
			cmdArgs = append(cmdArgs, cmdline...)
			cmdArgs = append(cmdArgs, assetUrls...)
			output, err := well.CommandContext(ctx, neco.CKECLIBin, cmdArgs...).Output()

			mtx.Lock()
			defer mtx.Unlock()
			if err != nil {
				fmt.Print(string(output))
				failedAddrs = append(failedAddrs, addr)
			} else {
				succeededAddrs = append(succeededAddrs, addr)
				succeededMachines = append(succeededMachines, machine)
			}
		}(machine)
	}
	wg.Wait()

	sort.Strings(succeededAddrs)
	sort.Strings(failedAddrs)

	fmt.Printf(`
Total:     %d
Succeeded: %d %v
Failed:    %d %v

`, len(machines), len(succeededAddrs), succeededAddrs, len(failedAddrs), failedAddrs)

	if !needReboot {
		return
	}

	ans, err := input.DefaultUI().Ask("reboot succeeded machines? ([y]es/[N]o)", &input.Options{
		Default:     "N",
		HideDefault: true,
		HideOrder:   true,
	})
	if err != nil {
		log.ErrorExit(err)
	}
	switch ans {
	case "y", "Y", "yes":
	default:
		return
	}

	err = rebootMachines(ctx, succeededMachines)
	if err != nil {
		log.ErrorExit(err)
	}
}

func getInterfaceAddress(ifname string) (net.IP, error) {
	link, err := netlink.LinkByName(ifname)
	if err != nil {
		return nil, err
	}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	return addrs[0].IP, nil
}

func init() {
	rootCmd.AddCommand(applyFirmwareCmd)
	addSabakanMachinesGetOpts(applyFirmwareCmd, &applyFirmwareGetOpts)
	applyFirmwareCmd.Flags().BoolVar(&applyFirmwareRebootOption, "reboot", false, "Schedule reboot")
}
