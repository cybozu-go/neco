package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var serfTagSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "set a configuration value to serf tag",
	Long: `Set a configuration value to serf tag.

Possible keys are:
    proxy - proxy url.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("accepts %d arg(s), received %d", 1, len(args))
		}
		switch args[0] {
		case "proxy":
			if len(args) != 2 {
				return fmt.Errorf("accepts %d arg(s), received %d", 2, len(args))
			}
		}
		return nil
	},
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
		var value string
		well.Go(func(ctx context.Context) error {
			switch key {
			case "proxy":
				value = args[1]
				u, err := url.Parse(value)
				if err != nil {
					return err
				}
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				return st.PutSerfTagProxy(ctx, value)
			}
			return errors.New("unknown key: " + key)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	serfTagCmd.AddCommand(serfTagSetCmd)
}
