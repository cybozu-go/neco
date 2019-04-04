package cmd

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var rebootWorkerCmd = &cobra.Command{
	Use:   "reboot-worker",
	Short: "Reboot all worker nodes.",
	Long:  `Reboot all worker nodes for their updates.`,

	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		saba, err := sabakan.NewClient(neco.SabakanLocalEndpoint, ext.LocalHTTPClient())
		if err != nil {
			return
		}
		params := make(map[string]string)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		machines, err := saba.MachinesGet(ctx, params)
		if err != nil {
			log.ErrorExit(err)
		}
		var workerName string
		data, _ := ioutil.ReadFile("/sys/devices/virtual/dmi/id/sys_vendor")
		if strings.TrimSpace(string(data)) == "QEMU" {
			workerName = "worker"
		} else {
			workerName = "cs"
		}
		var serials []string
		for _, m := range machines {
			if m.Spec.Role == workerName {
				serials = append(serials, m.Spec.Serial)
			}
		}

		driverVersion := getDriver()
		for _, s := range serials {
			well.Go(func(ctx context.Context) error {
				addr, err := lookupMachineBMCAddress(ctx, s)
				if err != nil {
					return err
				}
				return ipmiPower(ctx, "restart", driverVersion, addr)
			})
		}
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(rebootWorkerCmd)
}
