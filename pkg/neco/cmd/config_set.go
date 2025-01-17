package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
)

// configSetCmd implements "neco config set"
var configSetCmd = &cobra.Command{
	Use:   "set KEY [VALUE]",
	Short: "store a configuration value to etcd",
	Long: `Store a configuration value to etcd.

Possible keys are:
    env                          - "staging" or "prod".
    slack                        - Slack WebHook URL.
    proxy                        - HTTP proxy server URL to access Internet for boot servers.
    check-update-interval        - Polling interval for checking new neco release.
    worker-timeout               - Timeout value to wait for workers.
    github-token                 - GitHub personal access token for checking GitHub release.
    node-proxy                   - HTTP proxy server URL to access Internet for worker nodes.
    external-ip-address-block    - IP address block to be assigned to Nodes by LoadBalancer controllers.
	lb-address-block-default     - LoadBalancer address block for default.
	lb-address-block-bastion     - LoadBalancer address block for bastion.
	lb-address-block-internet    - LoadBalancer address block for internet.
	lb-address-block-internet-cn - LoadBalancer address block for internet-cn.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("accepts %d arg(s), received %d", 1, len(args))
		}
		switch args[0] {
		case "env", "slack", "proxy", "check-update-interval", "worker-timeout", "node-proxy", "external-ip-address-block":
			if len(args) != 2 {
				return fmt.Errorf("accepts %d arg(s), received %d", 2, len(args))
			}
		}
		return nil
	},
	ValidArgs: []string{
		"env",
		"slack",
		"proxy",
		"check-update-interval",
		"worker-timeout",
		"github-token",
		"node-proxy",
		"external-ip-address-block",
		"release-time",
		"release-timezone",
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
				if value != neco.TestEnv && value != neco.DevEnv && value != neco.StagingEnv && value != neco.ProdEnv {
					return errors.New("invalid environment")
				}
				return st.PutEnvConfig(ctx, value)
			case "slack":
				value = args[1]
				u, err := url.Parse(value)
				if err != nil {
					return err
				}
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				return st.PutSlackNotification(ctx, value)
			case "proxy":
				value = args[1]
				u, err := url.Parse(value)
				if err != nil {
					return err
				}
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				return st.PutProxyConfig(ctx, value)
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
			case "github-token":
				value = args[1]
				return st.PutGitHubToken(ctx, value)
			case "node-proxy":
				value = args[1]
				u, err := url.Parse(value)
				if err != nil {
					return err
				}
				if !u.IsAbs() {
					return errors.New("invalid URL")
				}
				return st.PutNodeProxy(ctx, value)
			case "external-ip-address-block":
				value = args[1]
				ip, block, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				if ip.To4() == nil {
					return errors.New("not IPv4 addr: " + value)
				}
				return st.PutExternalIPAddressBlock(ctx, block.String())
			case "lb-address-block-default":
				value = args[1]
				ip, block, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				if ip.To4() == nil {
					return errors.New("not IPv4 addr: " + value)
				}
				return st.PutLBAddressBlockDefault(ctx, block.String())
			case "lb-address-block-bastion":
				value = args[1]
				ip, block, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				if ip.To4() == nil {
					return errors.New("not IPv4 addr: " + value)
				}
				return st.PutLBAddressBlockBastion(ctx, block.String())
			case "lb-address-block-internet":
				value = args[1]
				ip, block, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				if ip.To4() == nil {
					return errors.New("not IPv4 addr: " + value)
				}
				return st.PutLBAddressBlockInternet(ctx, block.String())
			case "lb-address-block-internet-cn":
				value = args[1]
				ip, block, err := net.ParseCIDR(value)
				if err != nil {
					return err
				}
				if ip.To4() == nil {
					return errors.New("not IPv4 addr: " + value)
				}
				return st.PutLBAddressBlockInternetCN(ctx, block.String())
			case "release-time":
				values := args[1:]
				for _, value := range values {
					_, err := cron.ParseStandard(value)
					if err != nil {
						return err
					}
				}
				value = strings.Join(values, ",")
				return st.PutReleaseTime(ctx, value)
			case "release-timezone":
				value = args[1]
				_, err := time.LoadLocation(value)
				if err != nil {
					return err
				}
				return st.PutReleaseTimeZone(ctx, value)
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
