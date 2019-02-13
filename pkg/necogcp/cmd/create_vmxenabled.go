package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var createVMXEnabledCommand = &cobra.Command{
	Use:   "vmx-enabled",
	Short: "Create vmx-enabled image",
	Long: `Create vmx-enabled image.

If vmx-enabled image already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "vmx-enabled")
		well.Go(func(ctx context.Context) error {
			cc.DeleteInstance(ctx)

			err := cc.CreateVMXEnabledInstance(ctx)
			if err != nil {
				return err
			}

			err = cc.WaitInstance(ctx)
			if err != nil {
				return err
			}

			progFile, err := os.Executable()
			if err != nil {
				return err
			}

			err = cc.RunSetup(ctx, progFile, cfgFile)
			if err != nil {
				return err
			}

			err = cc.StopInstance(ctx)
			if err != nil {
				return err
			}

			cc.DeleteVMXEnabledImage(ctx)

			err = cc.CreateVMXEnabledImage(ctx)
			if err != nil {
				return err
			}

			err = cc.DeleteInstance(ctx)
			if err != nil {
				return err
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

func init() {
	createCmd.AddCommand(createVMXEnabledCommand)
}
