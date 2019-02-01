package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func getMyMachine(ctx context.Context) (*sabakan.Machine, error) {
	data, err := ioutil.ReadFile("/sys/devices/virtual/dmi/id/product_serial")
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

func setupHW(ctx context.Context, st storage.Storage) error {
	my, err := getMyMachine(ctx)
	if err != nil {
		return err
	}

	bmcAddr, err := os.OpenFile("/etc/neco/bmc-address.json", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer bmcAddr.Close()

	err = json.NewEncoder(bmcAddr).Encode(my.Info.BMC)
	if err != nil {
		return err
	}

	bmcUser, err := st.GetBMCBMCUser(ctx)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("/etc/neco/bmc-user.json", []byte(bmcUser), 0644)
	if err != nil {
		return err
	}

	c, err := neco.EnterContainerAppCommand(ctx, "setup-hw", []string{"setup-hw"})
	if err != nil {
		return err
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Run()
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		log.ErrorExit(err)
	}
	status := exitErr.Sys().(syscall.WaitStatus)
	// If status code is 10, the server need to be rebooted.
	// https://github.com/cybozu-go/setup-hw/blob/master/README.md#how-to-run-setup-hw
	if status.ExitStatus() == 10 {
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
