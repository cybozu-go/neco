package cmd

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "store a configuration value to etcd",
	Long: `Store a configuration value to etcd.

Possible keys are:
    env                   - "staging" or "prod".  Default is "staging".
    slack                 - Slack WebHook URL.
    proxy                 - HTTP proxy server URL to access Internet.
    check-update-interval - Polling interval for checking new neco release.
    worker-timeout        - Timeout value to wait for workers.`,

	Args: cobra.ExactArgs(2),
	ValidArgs: []string{
		"env",
		"slack",
		"proxy",
		"check-update-interval",
		"worker-timeout",
	},
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		key := args[0]
		value := args[1]
		well.Go(func(ctx context.Context) error {
			switch key {
			case "env":
				if value != neco.StagingEnv && value != neco.ProdEnv {
					return errors.New("invalid environment")
				}
				return st.PutEnvConfig(ctx, value)
			case "slack":
				_, err := url.Parse(value)
				if err != nil {
					return err
				}
				return st.PutSlackNotification(ctx, value)
			case "proxy":
				_, err := url.Parse(value)
				if err != nil {
					return err
				}
				return st.PutProxyConfig(ctx, value)
			case "check-update-interval":
				duration, err := time.ParseDuration(value)
				if err != nil {
					return err
				}
				return st.PutCheckUpdateInterval(ctx, duration)
			case "worker-timeout":
				duration, err := time.ParseDuration(value)
				if err != nil {
					return err
				}
				return st.PutWorkerTimeout(ctx, duration)
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
	configCmd.AddCommand(configSetCmd)
}
