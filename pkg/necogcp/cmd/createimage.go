package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var createImageCommand = &cobra.Command{
	Use:   "create-image",
	Short: "Create vmx-enabled image",
	Long: `Create vmx-enabled image.

If vmx-enabled image already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "vmx-enabled")
		well.Go(func(ctx context.Context) error {
			return gcp.CreateVMXEnabledImage(ctx, cc, cfgFile)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(createImageCommand)
}
