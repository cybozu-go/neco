package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var serfTagGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "get a configuration value from etcd",
	Long: `Get a configuration value from etcd.

Possible keys are:
    proxy - proxy url.`,

	Args: cobra.ExactArgs(1),
	ValidArgs: []string{
		"proxy",
	},
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		key := args[0]
		well.Go(func(ctx context.Context) error {
			switch key {
			case "proxy":
				proxy, err := st.GetSerfTagProxy(ctx)
				if err != nil {
					return err
				}
				fmt.Println(proxy)
			default:
				return errors.New("unknown key: " + key)
			}
			return nil
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	serfTagCmd.AddCommand(serfTagGetCmd)
}
