package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcdpasswd"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var initLocalParams struct {
	name string
}

// initLocalCmd represents the initLocal command
var initLocalCmd = &cobra.Command{
	Use:   "init-local NAME",
	Short: "Initialize data for new application of a boot server executes",
	Long: `Initialize data for new application of a boot server executes. This
command should not be executed more than once.  It asks vault user and
password to generate a vault token, then issue client certificates for
new a application NAME.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		initLocalParams.name = args[0]
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		mylrn, err := neco.MyLRN()
		if err != nil {
			log.ErrorExit(err)
		}
		vc, err := vaultClient(mylrn)
		if err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			var err error
			switch initLocalParams.name {
			case "etcdpasswd":
				err = etcdpasswd.IssueCerts(ctx, vc)
			case "sabakan":
				err = sabakan.IssueCerts(ctx, vc)
			default:
				return errors.New("unknown service name: " + initLocalParams.name)
			}
			if err != nil {
				return err
			}

			switch initLocalParams.name {
			case "etcdpasswd":
				err = neco.StartService(ctx, neco.EtcdpasswdService)
			case "sabakan":
				err = neco.StartService(ctx, neco.SabakanService)
			}
			return err
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initLocalCmd)
}
