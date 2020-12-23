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
    env                       - "staging" or "prod".  Default is "staging".
    slack                     - Slack WebHook URL.
    proxy                     - HTTP proxy server URL to access Internet for boot servers.
    dns                       - DNS server address for boot servers.
    quay-username             - Username to authenticate to quay.io.
    check-update-interval     - Polling interval for checking new neco release.
    worker-timeout            - Timeout value to wait for workers.
    github-token              - GitHub personal access token for checking GitHub release.
    node-proxy                - HTTP proxy server URL to access Internet for worker nodes.
    external-ip-address-block - IP address block to be assigned to Nodes by LoadBalancer controllers.
	registry                  - Registry mirror server URL for containerd.`,

	Args: cobra.ExactArgs(1),
	ValidArgs: []string{
		"env",
		"slack",
		"proxy",
		"dns",
		"quay-username",
		"check-update-interval",
		"worker-timeout",
		"github-token",
		"node-proxy",
		"external-ip-address-block",
		"registry",
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
			case "dns":
				dns, err := st.GetDNSConfig(ctx)
				if err != nil {
					return err
				}
				fmt.Println(dns)
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
			case "registry":
				registries, err := st.ListRegistryConfig(ctx)
				if err != nil {
					return err
				}
				for k, v := range registries {
					fmt.Printf("%s: %s\n", k, v)
				}
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
