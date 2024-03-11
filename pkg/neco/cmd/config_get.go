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

// configGetCmd implements "neco config get"
var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "show the current configuration value",
	Long: `Show the current configuration value.

Possible keys are:
    env                          - "staging" or "prod".  Default is "staging".
    slack                        - Slack WebHook URL.
    proxy                        - HTTP proxy server URL to access Internet for boot servers.
    quay-username                - Username to authenticate to quay.io.
    check-update-interval        - Polling interval for checking new neco release.
    worker-timeout               - Timeout value to wait for workers.
    github-token                 - GitHub personal access token for checking GitHub release.
    node-proxy                   - HTTP proxy server URL to access Internet for worker nodes.
    external-ip-address-block    - IP address block to be assigned to Nodes by LoadBalancer controllers.
    lb-address-block-default     - LoadBalancer address block for default.
    lb-address-block-bastion     - LoadBalancer address block for bastion.
    lb-address-block-internet    - LoadBalancer address block for internet.
    lb-address-block-internet-cn - LoadBalancer address block for internet-cn.`,

	Args: cobra.ExactArgs(1),
	ValidArgs: []string{
		"env",
		"slack",
		"proxy",
		"quay-username",
		"check-update-interval",
		"worker-timeout",
		"github-token",
		"node-proxy",
		"external-ip-address-block",
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
			case "env":
				env, err := st.GetEnvConfig(ctx)
				if err != nil {
					return err
				}
				fmt.Println(env)
			case "slack":
				slack, err := st.GetSlackNotification(ctx)
				if err != nil {
					return err
				}
				fmt.Println(slack)
			case "proxy":
				proxy, err := st.GetProxyConfig(ctx)
				if err != nil {
					return err
				}
				fmt.Println(proxy)
			case "quay-username":
				username, err := st.GetQuayUsername(ctx)
				if err != nil {
					return err
				}
				fmt.Println(username)
			case "check-update-interval":
				interval, err := st.GetCheckUpdateInterval(ctx)
				if err != nil {
					return err
				}
				fmt.Println(interval.String())
			case "worker-timeout":
				timeout, err := st.GetWorkerTimeout(ctx)
				if err != nil {
					return err
				}
				fmt.Println(timeout.String())
			case "github-token":
				token, err := st.GetGitHubToken(ctx)
				if err != nil {
					return err
				}
				fmt.Println(token)
			case "node-proxy":
				proxy, err := st.GetNodeProxy(ctx)
				if err != nil {
					return err
				}
				fmt.Println(proxy)
			case "external-ip-address-block":
				ipBlock, err := st.GetExternalIPAddressBlock(ctx)
				if err != nil {
					return err
				}
				fmt.Println(ipBlock)
			case "lb-address-block-default":
				ipBlock, err := st.GetLBAddressBlockDefault(ctx)
				if err != nil {
					return err
				}
				fmt.Println(ipBlock)
			case "lb-address-block-bastion":
				ipBlock, err := st.GetLBAddressBlockBastion(ctx)
				if err != nil {
					return err
				}
				fmt.Println(ipBlock)
			case "lb-address-block-internet":
				ipBlock, err := st.GetLBAddressBlockInternet(ctx)
				if err != nil {
					return err
				}
				fmt.Println(ipBlock)
			case "lb-address-block-internet-cn":
				ipBlock, err := st.GetLBAddressBlockInternetCN(ctx)
				if err != nil {
					return err
				}
				fmt.Println(ipBlock)
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
	configCmd.AddCommand(configGetCmd)
}
