package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var initParams struct {
	name string
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init NAME",
	Short: "Initialize data for new application of the cluster",
	Long: `Initialize data for new application of the cluster.
Setup etcd user/role for a new application NAME. This command should not be
executed more than once.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		initParams.name = args[0]
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		ce, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ce.Close()

		well.Go(func(ctx context.Context) error {
			switch initParams.name {
			case "etcdpasswd":
				return etcd.UserAdd(ctx, ce, "etcdpasswd", neco.EtcdpasswdPrefix)
			case "sabakan":
				return etcd.UserAdd(ctx, ce, "sabakan", neco.SabakanPrefix)
			default:
				return errors.New("unknown service name: " + initParams.name)
			}
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
