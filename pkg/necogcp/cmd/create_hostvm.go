package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var createHostVMCommand = &cobra.Command{
	Use:   "host-vm",
	Short: "Launch host-vm instance",
	Long: `Launch host-vm instance using vmx-enabled image.

If host-vm instance already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			cc.DeleteInstance(ctx)

			err := cc.CreateHostVMInstance(ctx)
			if err != nil {
				return err
			}

			err = cc.WaitInstance(ctx)
			if err != nil {
				return err
			}

			err = cc.CreateHomeDisk(ctx)
			if err != nil {
				return err
			}

			err = cc.ResizeHomeDisk(ctx)
			if err != nil {
				return err
			}

			err = cc.AttachHomeDisk(ctx)
			if err != nil {
				return err
			}

			progFile, err := os.Executable()
			if err != nil {
				return err
			}

			return cc.RunSetup(ctx, progFile, cfgFile)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		fmt.Println("host-vm has been created! Ready to login")
	},
}

func init() {
	createCmd.AddCommand(createHostVMCommand)
}
