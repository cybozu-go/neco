package cmd

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	sabakan "github.com/cybozu-go/sabakan/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func lookupMachineBMCAddress(ctx context.Context, id string) (string, error) {
	ip := net.ParseIP(id)
	params := make(map[string]string)
	if ip != nil {
		if ip.To4() != nil {
			params["ipv4"] = id
		} else {
			params["ipv6"] = id
		}
	} else {
		params["serial"] = id
	}

	saba, err := sabakan.NewClient(neco.SabakanLocalEndpoint, ext.LocalHTTPClient())
	if err != nil {
		return "", err
	}

	machines, err := saba.MachinesGet(ctx, params)
	if err != nil {
		return "", err
	}

	if len(machines) != 1 {
		return "", errors.New("machine is not found in sabakan")
	}

	return machines[0].Spec.BMC.IPv4, nil
}

func ipmiPower(ctx context.Context, action, addr string) error {
	var opts []string
	switch action {
	case "start":
		opts = append(opts, "--on", "--wait-until-on")
	case "stop":
		opts = append(opts, "--off", "--wait-until-off")
	case "restart":
		opts = append(opts, "--reset")
	case "status":
		opts = append(opts, "--stat")
	default:
		return errors.New("invalid action: " + action)
	}

	etcd, err := neco.EtcdClient()
	if err != nil {
		return err
	}
	defer etcd.Close()
	st := storage.NewStorage(etcd)

	username, err := st.GetBMCIPMIUser(ctx)
	if err != nil {
		return err
	}
	password, err := st.GetBMCIPMIPassword(ctx)
	if err != nil {
		return err
	}

	args := append(opts, "-u", username, "-p", password, "-h", addr)
	cmd := exec.CommandContext(ctx, "ipmipower", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var ipmiPowerCmd = &cobra.Command{
	Use:     "ipmipower ACTION SERIAL|IP",
	Aliases: []string{"power"},
	Short:   "control power of a machine",
	Long: `Control power of a machine using ipmipower command.
	
	ACTION should be one of:
		- start:   to turn on the machine power.
		- stop:    to turn off the machine power.
		- restart: to hard reset the machine.
		- status:  to report the power status of the machine.
		
	SERIAL is the serial number of the machine.
	IP is one of the IP addresses owned by the machine.`,

	Args:      cobra.ExactArgs(2),
	ValidArgs: []string{"start", "stop", "restart", "status"},
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			addr, err := lookupMachineBMCAddress(ctx, args[1])
			if err != nil {
				return err
			}

			return ipmiPower(ctx, args[0], addr)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(ipmiPowerCmd)
}
