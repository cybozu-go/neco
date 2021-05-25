package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/clientv3"
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
			case "cke":
				return etcd.UserAdd(ctx, ce, "cke", neco.CKEPrefix)
			case "teleport":
				return generateAndSetToken(ctx, ce, 32)
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

func generateAndSetToken(ctx context.Context, ce *clientv3.Client, bytes int) error {
	buf := make([]byte, bytes)
	_, err := rand.Read(buf)
	if err != nil {
		return err
	}
	token := hex.EncodeToString(buf)

	st := storage.NewStorage(ce)
	return st.PutTeleportAuthToken(ctx, token)
}
