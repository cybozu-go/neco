package cmd

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func setupHW(ctx context.Context, st storage.Storage) error {
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
