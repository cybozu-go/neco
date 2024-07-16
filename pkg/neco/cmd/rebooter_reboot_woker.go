package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/cybozu-go/sabakan/v3/client"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/tcnksm/go-input"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	rebootWorkerGetOpts sabakanMachinesGetOpts
	flagDryRun          bool
)

// sabakanMachinesGetOpts is a struct to receive option values for `sabactl machines get`-like options
type sabakanMachinesGetOpts struct {
	params map[string]*string
}

var rebooterRebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "reboot all worker nodes",
	Long: `Reboot all worker nodes for their updates.

This uses neco-rebooter and CKE's function of graceful reboot for the nodes used by CKE.
As for the other nodes, this reboots them immediately.
If some nodes are already powered off, this command does not do anything to those nodes.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("WARNING: this command reboots all servers other than boot servers and may cause system instability.")
		ans, err := askYorN("Continue?")
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
		fmt.Println("WARNING: rebooting starts immediately after this question.")
		ans, err = askYorN("Continue?")
		if err != nil {
			return err
		}
		if !ans {
			return nil
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		ckeDisabled, err := ckeStorage.IsRebootQueueDisabled(ctx)
		if err != nil {
			return err
		}
		if ckeDisabled {
			fmt.Println("WARNING: CKE reboot queue is disabled.")
		}

		machines, err := sabakanMachinesGet(ctx, &rebootWorkerGetOpts)
		if err != nil {
			return err
		}
		err = rebootMachines(ctx, machines)
		if err != nil {
			return err
		}
		return nil
	},
}

func rebootMachines(ctx context.Context, machines []sabakan.Machine) error {
	retryCount := 0
RETRY:
	nodes, err := KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		if retryCount > 2 {
			return err
		}
		err := renewKubeConfig()
		if err != nil {
			return err
		}
		err = loadKubeConfig()
		if err != nil {
			return err
		}
		retryCount++
		goto RETRY
	}

	nonKubernetesNodeMachine := make([]sabakan.Machine, 0)
	kubernetesNodeMachine := make([]corev1.Node, 0)

	kubernetesNodeAddrs := make(map[string]corev1.Node, len(nodes.Items))
	for _, node := range nodes.Items {
		kubernetesNodeAddrs[node.Name] = node
	}
	for _, machine := range machines {
		if len(machine.Spec.IPv4) == 0 {
			slog.Warn("IP addresses not found; skipping", "serial", machine.Spec.Serial)
			continue
		}
		if node, ok := kubernetesNodeAddrs[machine.Spec.IPv4[0]]; ok {
			kubernetesNodeMachine = append(kubernetesNodeMachine, node)
		} else {
			nonKubernetesNodeMachine = append(nonKubernetesNodeMachine, machine)
		}
	}

	//reboot kubernetes nodes
	for _, node := range kubernetesNodeMachine {
		group, ok := node.ObjectMeta.Labels[config.GroupLabelKey]
		if !ok {
			return fmt.Errorf("node has no groupKey label (%s)", config.GroupLabelKey)
		}
		rt, err := matchRebootTimes(node, config.RebootTimes)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("adding a node to reboot list node=%s  group=%s reboot_time=%s\n", node.Name, group, rt.Name)
		newEntry := neco.RebootListEntry{
			Node:       node.Name,
			Group:      group,
			RebootTime: rt.Name,
			Status:     neco.RebootListEntryStatusPending,
		}
		if !flagDryRun {
			err := necoStorage.RegisterRebootListEntry(ctx, &newEntry)
			if err != nil {
				return err
			}
		}
	}
	// reboot non-kubernetes nodes immediately
	var errorNodes []string
	for _, m := range nonKubernetesNodeMachine {
		if m.Spec.Role == "boot" {
			continue
		}
		if len(m.Spec.IPv4) == 0 {
			slog.Warn("IP addresses not found; skipping", "serial", m.Spec.Serial)
			continue
		}
		addr := m.Spec.BMC.IPv4
		if addr == "" {
			slog.Warn("BMC IP address not found; skipping", "serial", m.Spec.Serial, "node", m.Spec.IPv4[0])
			continue
		}
		slog.Info("rebooting node", "serial", m.Spec.Serial, "node", m.Spec.IPv4[0], "bmc", m.Spec.BMC.IPv4)
		if !flagDryRun {
			err := rebootNode(ctx, addr)
			if err != nil {
				slog.Warn("failed to restart node", "serial", m.Spec.Serial, "node", m.Spec.IPv4[0], "err", err)
				errorNodes = append(errorNodes, m.Spec.Serial)
			}
		}
	}
	if len(errorNodes) != 0 {
		return fmt.Errorf("failed to reboot on some nodes: %s", strings.Join(errorNodes, ","))
	}
	return nil
}

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
	c, err := client.NewClient(neco.SabakanLocalEndpoint, &http.Client{})
	if err != nil {
		return nil, err
	}
	return c.MachinesGet(ctx, params)
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
	rebooterRebootWorkerCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "dry-run")
	rebooterCmd.AddCommand(rebooterRebootWorkerCmd)
	addSabakanMachinesGetOpts(rebooterRebootWorkerCmd, &rebootWorkerGetOpts)

}
