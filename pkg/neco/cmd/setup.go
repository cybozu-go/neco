package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/setup"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var setupParams struct {
	lrns        []int
	noRevoke    bool
	proxy       string
	ghTokenFile string
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup LRN [LRN ...]",
	Short: "Install and setup etcd cluster as well as Vault using given boot servers",
	Long: `Install and setup etcd cluster as well as Vault using given boot
servers. LRN is the logical rack number of the boot server. At least 3
LRNs should be specified.

This command should be invoked at once on all boot servers specified by LRN.

When --no-revoke option is specified, it does not remove the etcd key
<prefix>/vault-root-token. This option is used by automatic setup of dctest.

When --gh-token-file option is specified, it will read a GitHub token and store
it in etcd. If not, it will read it from stdin instead. The stored token will used
by neco-worker when it sends requests to GitHub API to avoid the rate limit exceeded error.

When --proxy option is specified, it uses this proxy to download container
images. It also stores proxy configuration in the etcd database after it
starts etcd, in order to run neco-updater and neco-worker with a proxy
from the start.
DO NOT pass http_proxy and https_proxy environment variables to neco.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return errors.New("too few boot servers")
		}
		setupParams.lrns = make([]int, len(args))
		for i, a := range args {
			num, err := strconv.ParseUint(a, 10, 32)
			if err != nil {
				return err
			}
			setupParams.lrns[i] = int(num)
		}

		if len(setupParams.proxy) > 0 {
			u, err := url.Parse(setupParams.proxy)
			if err != nil {
				return err
			}
			if !u.IsAbs() {
				return errors.New("proxy not absolute")
			}
		}

		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		input := os.Stdin
		if len(setupParams.ghTokenFile) > 0 {
			f, err := os.Open(setupParams.ghTokenFile)
			if err != nil {
				log.ErrorExit(err)
			}
			defer f.Close()
			input = f
		}
		token, err := io.ReadAll(input)
		if err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			return setup.Setup(ctx, setupParams.lrns, !setupParams.noRevoke, setupParams.proxy, string(bytes.TrimSpace(token)))
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().BoolVar(&setupParams.noRevoke, "no-revoke", false, "keep vault root token in etcd")
	setupCmd.Flags().StringVar(&setupParams.proxy, "proxy", "", "use and store config of HTTP proxy server")
	setupCmd.Flags().StringVar(&setupParams.ghTokenFile, "gh-token-file", "", "gh token file for neco-worker")
}
