package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan/v2"
	sabaclient "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

func lookupMachine(ctx context.Context, id string) (*sabakan.Machine, error) {
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

	saba, err := sabaclient.NewClient(neco.SabakanLocalEndpoint, ext.LocalHTTPClient())
	if err != nil {
		return nil, err
	}

	machines, err := saba.MachinesGet(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(machines) != 1 {
		return nil, errors.New("machine is not found in sabakan")
	}

	return &machines[0], nil
}

func getBMCUsernameAndPassword(ctx context.Context) (string, string, error) {
	etcd, err := neco.EtcdClient()
	if err != nil {
		return "", "", err
	}
	defer etcd.Close()
	st := storage.NewStorage(etcd)

	username, err := st.GetBMCIPMIUser(ctx)
	if err != nil {
		return "", "", err
	}
	password, err := st.GetBMCIPMIPassword(ctx)
	if err != nil {
		return "", "", err
	}
	return username, password, nil
}

func getRedfishClient(ctx context.Context, bmcAddr string) (*gofish.APIClient, error) {
	username, password, err := getBMCUsernameAndPassword(ctx)
	if err != nil {
		return nil, err
	}

	config := gofish.ClientConfig{
		Endpoint:  fmt.Sprintf("https://%s", bmcAddr),
		Username:  username,
		Password:  password,
		BasicAuth: true,
		Insecure:  true,
	}
	return gofish.Connect(config)
}

func getComputerSystem(service *gofish.Service) (*redfish.ComputerSystem, error) {
	systems, err := service.Systems()
	if err != nil {
		return nil, err
	}

	// Check if the collection contains 1 computer system
	if len(systems) != 1 {
		return nil, fmt.Errorf("computer Systems length should be 1, actual: %d", len(systems))
	}

	return systems[0], nil
}

func waitForStop(ctx context.Context, nodeAddr string) error {
	// It might be better to use any golang library instead of executing ping command...
	for {
		err := exec.CommandContext(ctx, "ping", "-c1", "-w1", nodeAddr).Run()
		exitError, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}
		if exitError.ExitCode() != 0 {
			return nil
		}

		// In case of exit status == 0, ping command return immediately. Must wait explicitly.
		time.Sleep(time.Millisecond * 500)
	}
}

func power(ctx context.Context, action, bmcAddr string) error {
	client, err := getRedfishClient(ctx, bmcAddr)
	if err != nil {
		return err
	}
	defer client.Logout()

	system, err := getComputerSystem(client.Service)
	if err != nil {
		return err
	}

	var resetType redfish.ResetType
	switch action {
	case "start":
		// Use 'ON' because some machines don't support 'ForceOn'.
		resetType = redfish.OnResetType
	case "stop":
		resetType = redfish.ForceOffResetType
	case "restart":
		// Use 'ForceRestart' because some machines don't support 'GracefulRestart'.
		resetType = redfish.ForceRestartResetType
	case "status":
		fmt.Println(system.PowerState)
		return nil
	default:
		return errors.New("invalid action: " + action)
	}

	err = system.Reset(resetType)
	if err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}

var powerCmd = &cobra.Command{
	Use:     "power ACTION SERIAL|IP",
	Aliases: []string{"ipmipower"},
	Short:   "control power of a machine",
	Long: `Control power of a machine using Redfish API.
	
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
			action := args[0]
			if (action != "stop" && action != "restart") && waitForStopFlag {
				return fmt.Errorf("invalid flag for %s action: --wait-for-stop", action)
			}

			machine, err := lookupMachine(ctx, args[1])
			if err != nil {
				return err
			}
			err = power(ctx, action, machine.Spec.BMC.IPv4)
			if err != nil {
				return err
			}
			if waitForStopFlag {
				err := waitForStop(ctx, machine.Spec.IPv4[0])
				if err != nil {
					return err
				}
			}
			return nil
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

var waitForStopFlag bool

func init() {
	rootCmd.AddCommand(powerCmd)
	powerCmd.Flags().BoolVar(&waitForStopFlag, "wait-for-stop", false, "")
}
