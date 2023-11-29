package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/cybozu-go/sabakan/v3/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/tcnksm/go-input"
	"sigs.k8s.io/yaml"
)

var httpClient = &well.HTTPClient{
	Client: &http.Client{},
}

// sabakanMachinesGetOpts is a struct to receive option values for `sabactl machines get`-like options
type sabakanMachinesGetOpts struct {
	params map[string]*string
}

// addSabapanMachinesGetOpts adds flags for `sabactl machines get`-like options to cobra.Command
func addSabakanMachinesGetOpts(cmd *cobra.Command, opts *sabakanMachinesGetOpts) {
	getOpts := map[string]string{
		"serial":           "Serial name(s) (--serial 001,002,003...)",
		"rack":             "Rack name(s) (--rack 1,2,3...)",
		"role":             "Role name(s) (--role boot,worker...)",
		"labels":           "Label name and value (--labels key=val,...)",
		"ipv4":             "IPv4 address(s) (--ipv4 10.0.0.1,10.0.0.2,10.0.0.3...)",
		"ipv6":             "IPv6 address(s) (--ipv6 aa::ff,bb::ff,cc::ff...)",
		"bmc-type":         "BMC type(s) (--bmc-type iDRAC-9,IPMI-2.0...)",
		"state":            "State(s) (--state retiring,uninitialized...)",
		"without-serial":   "without Serial name",
		"without-rack":     "without Rack name",
		"without-role":     "without Role name",
		"without-labels":   "without Label name and value (--labels key=val,...)",
		"without-ipv4":     "without IPv4 address",
		"without-ipv6":     "without IPv6 address",
		"without-bmc-type": "without BMC type",
		"without-state":    "without State",
	}
	opts.params = make(map[string]*string)
	for k, v := range getOpts {
		val := new(string)
		opts.params[k] = val
		cmd.Flags().StringVar(val, k, "", v)
	}
}

// sabakanMachinesGet does the same as `sabactl machines get`
func sabakanMachinesGet(ctx context.Context, opts *sabakanMachinesGetOpts) ([]sabakan.Machine, error) {
	params := make(map[string]string)
	for k, v := range opts.params {
		params[k] = *v
	}
	c, err := client.NewClient(neco.SabakanLocalEndpoint, httpClient.Client)
	if err != nil {
		return nil, err
	}
	return c.MachinesGet(ctx, params)
}

// ckeCluster is part of cke.Cluster in github.com/cybozu-go/cke
type ckeCluster struct {
	Nodes []*ckeNode `yaml:"nodes"`
}

// ckeNode is part of cke.Node in github.com/cybozu-go/cke
type ckeNode struct {
	Address string            `yaml:"address"`
	Labels  map[string]string `yaml:"labels"`
}

var rebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "reboot all worker nodes",
	Long: `Reboot all worker nodes for their updates.

This uses CKE's function of graceful reboot for the nodes used by CKE.
As for the other nodes, this reboots them immediately.
If some nodes are already powered off, this command does not do anything to those nodes.`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("WARNING: this command reboots all servers other than boot servers and may cause system instability.")
		ans, err := askYorN("Continue?")
		if err != nil {
			log.ErrorExit(err)
		}
		if !ans {
			return
		}
		fmt.Println("WARNING: rebooting starts immediately after this question.")
		ans, err = askYorN("Continue?")
		if err != nil {
			log.ErrorExit(err)
		}
		if !ans {
			return
		}

		err = rebootMain()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

var rebootWorkerGetOpts sabakanMachinesGetOpts

func rebootMain() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	machines, err := sabakanMachinesGet(ctx, &rebootWorkerGetOpts)
	if err != nil {
		return err
	}

	toreboot := []sabakan.Machine{}
	for _, m := range machines {
		if m.Spec.Role != "boot" {
			toreboot = append(toreboot, m)
		}
	}

	return rebootMachines(toreboot)
}

func rebootMachines(machines []sabakan.Machine) error {
	machineAddrs := make(map[string]bool, len(machines))
	for _, m := range machines {
		if m.Spec.Role == "boot" {
			return fmt.Errorf("it's not allowed to reboot boot servers")
		}
		machineAddrs[m.Spec.IPv4[0]] = true
	}

	comm := exec.Command(neco.CKECLIBin, "cluster", "get")
	comm.Stderr = os.Stderr
	output, err := comm.Output()
	if err != nil {
		return err
	}
	cluster := new(ckeCluster)
	err = yaml.Unmarshal(output, cluster)
	if err != nil {
		return err
	}

	// group nodes by rack and role
	racks := map[string]bool{}
	ss := map[string][]string{}
	nonss := map[string][]string{}
	roles := map[string]string{}
	for _, node := range cluster.Nodes {
		if !machineAddrs[node.Address] {
			continue
		}

		rack := node.Labels["cke.cybozu.com/rack"]
		role := node.Labels["cke.cybozu.com/role"]

		racks[rack] = true
		roles[node.Address] = role
		if role == "ss" {
			ss[rack] = append(ss[rack], node.Address)
		} else {
			nonss[rack] = append(nonss[rack], node.Address)
		}
	}

	// enqueue in-cluster nodes per rack
	for rack := range racks {
		nonss := nonss[rack]
		ss := ss[rack]

		nonssIndex := 0
		ssIndex := 0

		// enqueue nodes so that SSs and non-SSs alternate as much as possible
		for i := 0; i < len(ss)+len(nonss); i++ {
			var address string
			if nonssIndex*len(ss) <= ssIndex*len(nonss) && len(nonss) > 0 {
				address = nonss[nonssIndex]
				nonssIndex++
			} else {
				address = ss[ssIndex]
				ssIndex++
			}
			log.Info("adding a node to CKE reboot queue", map[string]interface{}{
				"node": address,
				"rack": rack,
				"role": roles[address],
			})
			comm := exec.Command(neco.CKECLIBin, "reboot-queue", "add", "-")
			comm.Stdin = strings.NewReader(address + "\n")
			comm.Stdout = os.Stdout
			comm.Stderr = os.Stderr
			err := comm.Run()
			if err != nil {
				return err
			}
		}
	}

	// reboot out-of-cluster nodes immediately
	var errorNodes []string
	ckeNodeAddrs := make(map[string]bool, len(cluster.Nodes))
	for _, node := range cluster.Nodes {
		ckeNodeAddrs[node.Address] = true
	}
	for _, m := range machines {
		if len(m.Spec.IPv4) == 0 {
			log.Warn("IP addresses not found; skipping", map[string]interface{}{
				"serial": m.Spec.Serial,
			})
			continue
		}
		if ckeNodeAddrs[m.Spec.IPv4[0]] {
			continue
		}
		addr := m.Spec.BMC.IPv4
		if addr == "" {
			log.Warn("BMC IP address not found; skipping", map[string]interface{}{
				"serial": m.Spec.Serial,
				"node":   m.Spec.IPv4[0],
			})
			continue
		}
		log.Info("rebooting node", map[string]interface{}{
			"serial": m.Spec.Serial,
			"node":   m.Spec.IPv4[0],
			"bmc":    m.Spec.BMC.IPv4,
		})
		err := rebootNode(context.Background(), addr)
		if err != nil {
			log.Warn("failed to restart node", map[string]interface{}{
				"serial":    m.Spec.Serial,
				"node":      m.Spec.IPv4[0],
				log.FnError: err,
			})
			errorNodes = append(errorNodes, m.Spec.Serial)
		}
	}
	if len(errorNodes) != 0 {
		return fmt.Errorf("failed to reboot on some nodes: %s", strings.Join(errorNodes, ","))
	}
	return nil
}

func rebootNode(ctx context.Context, bmdAddr string) error {
	client, err := getRedfishClient(ctx, bmdAddr)
	if err != nil {
		return fmt.Errorf("failed to get redfish client: %s", err.Error())
	}
	defer client.Logout()

	system, err := getComputerSystem(client.Service)
	if err != nil {
		return fmt.Errorf("failed to get system instance: %s", err.Error())
	}

	if system.PowerState == redfish.OffPowerState {
		fmt.Println("skip: already powered OFF")
		return nil
	}

	err = system.Reset(redfish.ForceRestartResetType)
	if err != nil {
		return fmt.Errorf("failed to reset: %s", err.Error())
	}
	fmt.Println("ok")
	return nil
}

func askYorN(query string) (bool, error) {
	ans, err := input.DefaultUI().Ask(query+" [y/N]", &input.Options{
		Default:     "N",
		HideDefault: true,
		HideOrder:   true,
	})
	if err != nil {
		return false, err
	}
	switch ans {
	case "y", "Y", "yes", "YES":
		return true, nil
	}
	return false, nil
}

func init() {
	rootCmd.AddCommand(rebootWorkerCmd)
	addSabakanMachinesGetOpts(rebootWorkerCmd, &rebootWorkerGetOpts)
}
