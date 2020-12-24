package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var isRunningCmd = &cobra.Command{
	Use:   "is-running IMAGE",
	Short: "check if a container of IMAGE is running or not",
	Long: `This command exits with status 0 if a container whose image is IMAGE
is currently running.  If not, this exits with 1.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			return isRunning(ctx, args[0])
		})
		well.Stop()
		if err := well.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
			os.Exit(2)
		}
	},
}

func isRunning(ctx context.Context, name string) error {
	img, err := neco.CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}

	etcd, err := neco.EtcdClient()
	if err != nil {
		return err
	}
	defer etcd.Close()
	st := storage.NewStorage(etcd)
	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		return err
	}

	rt, err := neco.GetContainerRuntime(proxy)
	if err != nil {
		return err
	}

	ok, err := rt.IsRunning(img)
	if err != nil {
		return err
	}

	if !ok {
		fmt.Println("not running")
		os.Exit(1)
	}
	fmt.Println("running")
	return nil
}

func init() {
	rootCmd.AddCommand(isRunningCmd)
}
