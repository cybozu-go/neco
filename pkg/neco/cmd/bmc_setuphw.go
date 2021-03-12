package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan/v2"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func getMyMachine(ctx context.Context) (*sabakan.Machine, error) {
	data, err := os.ReadFile("/sys/devices/virtual/dmi/id/product_serial")
	if err != nil {
		return nil, err
	}
	serial := strings.TrimSpace(string(data))
	cmd := exec.CommandContext(ctx, "sabactl", "machines", "get", "--serial", serial)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var machines []*sabakan.Machine
	err = json.Unmarshal(out, &machines)
	if err != nil {
		return nil, err
	}

	if len(machines) != 1 {
		return nil, errors.New("no machine entry in sabakan")
	}

	return machines[0], nil
}

func writeBMCAddress(info interface{}) error {
	bmcAddr, err := os.OpenFile("/etc/neco/bmc-address.json", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer bmcAddr.Close()

	err = json.NewEncoder(bmcAddr).Encode(info)
	if err != nil {
		return err
	}

	return bmcAddr.Sync()
}

func setupHW(ctx context.Context, st storage.Storage) error {
	my, err := getMyMachine(ctx)
	if err != nil {
		return err
	}

	if err := writeBMCAddress(my.Info.BMC); err != nil {
		return fmt.Errorf("failed to write bmc-address.json: %w", err)
	}

	bmcUser, err := st.GetBMCBMCUser(ctx)
	if err != nil {
		return err
	}

	err = os.WriteFile("/etc/neco/bmc-user.json", []byte(bmcUser), 0644)
	if err != nil {
		return err
	}

	// this will soft-reset iDRAC
	err = neco.RestartService(ctx, "setup-hw")
	if err != nil {
		return err
	}

	err = well.CommandContext(ctx, "systemctl", "is-active", "setup-hw").Run()
	if err != nil {
		return err
	}

	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		return err
	}
	rt, err := neco.GetContainerRuntime(proxy)
	if err != nil {
		return err
	}

	err = neco.RetryWithSleep(ctx, 240, time.Second,
		func(ctx context.Context) error {
			return rt.Exec(ctx, "setup-hw", false, []string{"echo", "hello"})
		},
		func(err error) {
			//	set no logger rt.Exec logs own errors implicitly
		},
	)
	if err != nil {
		return err
	}

	// setup-hw container resets iDRAC at startup.
	// Wait here to avoid the reset from occurring while the tool is running.
	if err := exec.Command("systemd-detect-virt", "-q", "--vm").Run(); err != nil {
		log.Info("waiting for iDRAC reset...", nil)
		time.Sleep(1 * time.Minute)
	}
	err = rt.Exec(ctx, "setup-hw", true, []string{"setup-hw"})
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		log.ErrorExit(err)
	}

	// If status code is 10, the server need to be rebooted.
	// https://github.com/cybozu-go/setup-hw/blob/master/README.md#how-to-run-setup-hw
	if exitErr.ExitCode() == 10 {
		exec.Command("reboot").Run()
		return nil
	}

	return err
}

var bmcSetupHWCmd = &cobra.Command{
	Use:   "setup-hw",
	Short: "setup hardware",
	Long:  `Setup hardware.`,
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getuid() != 0 {
			log.ErrorExit(errors.New("run as root"))
		}

		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			return setupHW(ctx, st)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcCmd.AddCommand(bmcSetupHWCmd)
}
