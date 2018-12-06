package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var configSetCmd = &cobra.Command{
	Use:   "set KEY [VALUE]",
	Short: "store a configuration value to etcd",
	Long: `Store a configuration value to etcd.

Possible keys are:
    env                   - "staging" or "prod".  Default is "staging".
    slack                 - Slack WebHook URL.
    proxy                 - HTTP proxy server URL to access Internet.
    quay-username         - Username to authenticate to quay.io from QUAY_USER.  This does not take VALUE.
    quay-password         - Password to authenticate to quay.io from QUAY_PASSWORD.  This does not take VALUE.
    check-update-interval - Polling interval for checking new neco release.
    worker-timeout        - Timeout value to wait for workers.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("accepts %d arg(s), received %d", 1, len(args))
		}
		switch args[0] {
		case "env", "slack", "proxy", "check-update-interval", "worker-timeout":
			if len(args) != 2 {
				return fmt.Errorf("accepts %d arg(s), received %d", 2, len(args))
			}
		case "quay-password", "quay-username":
			if len(args) != 1 {
				return fmt.Errorf("accepts %d arg(s), received %d", 1, len(args))
			}
		}
		return nil
	},
	ValidArgs: []string{
		"env",
		"slack",
		"proxy",
		"quay-username",
		"quay-password",
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
		var value string
		well.Go(func(ctx context.Context) error {
			switch key {
			case "env":
				value = args[1]
				if value != neco.TestEnv && value != neco.StagingEnv && value != neco.ProdEnv {
					return errors.New("invalid environment")
				}
				return st.PutEnvConfig(ctx, value)
			case "slack":
				value = args[1]
				u, err := url.Parse(value)
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				if err != nil {
					return err
				}
				return st.PutSlackNotification(ctx, value)
			case "proxy":
				value = args[1]
				u, err := url.Parse(value)
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				if err != nil {
					return err
				}
				return st.PutProxyConfig(ctx, value)
			case "quay-username":
				value = os.Getenv("QUAY_USER")
				return st.PutQuayUsername(ctx, value)
			case "quay-password":
				value = os.Getenv("QUAY_PASSWORD")
				return st.PutQuayPassword(ctx, value)
			case "check-update-interval":
				value = args[1]
				duration, err := time.ParseDuration(value)
				if err != nil {
					return err
				}
				return st.PutCheckUpdateInterval(ctx, duration)
			case "worker-timeout":
				value = args[1]
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
