package cmd

import (
	"context"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/progs/cke"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var ignitionsOnly bool

func initData(ctx context.Context, st storage.Storage) error {
	version, err := neco.GetDebianVersion(neco.NecoPackageName)
	if err != nil {
		return err
	}

	proxyClient, err := ext.ProxyHTTPClient(ctx, st)
	if err != nil {
		return err
	}
	localClient := ext.LocalHTTPClient()

	// NOTE: hack for github.com/containers/image to set HTTP proxy
	proxy, err := st.GetProxyConfig(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	if len(proxy) != 0 {
		os.Setenv("http_proxy", proxy)
		os.Setenv("https_proxy", proxy)
	}

	username, err := st.GetQuayUsername(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	password, err := st.GetQuayPassword(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	var auth *sabakan.DockerAuth
	if len(username) != 0 && len(password) != 0 {
		auth = &sabakan.DockerAuth{
			Username: username,
			Password: password,
		}
	}

	if ignitionsOnly {
		return sabakan.UploadIgnitions(ctx, localClient, version, st)
	}

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return sabakan.UploadContents(ctx, localClient, proxyClient, version, auth, st)
	})
	env.Go(func(ctx context.Context) error {
		return cke.UploadContents(ctx, localClient, proxyClient, version)
	})
	env.Go(func(ctx context.Context) error {
		return sabakan.UploadDHCPJSON(ctx, localClient)
	})
	env.Go(func(ctx context.Context) error {
		return cke.SetCKETemplate(ctx, st)
	})
	env.Go(cke.UpdateResources)
	env.Stop()
	return env.Wait()
}

var initDataCmd = &cobra.Command{
	Use:   "init-data",
	Short: "initialize data for sabakan and CKE",
	Long: `Initialize data for sabakan and CKE
If uploaded versions are up to date, do nothing.
`,
	Run: func(cmd *cobra.Command, args []string) {
		ec, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ec.Close()
		st := storage.NewStorage(ec)

		well.Go(func(ctx context.Context) error {
			return initData(ctx, st)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	initDataCmd.Flags().BoolVar(&ignitionsOnly, "ignitions-only", false, "upload ignitions only")
	rootCmd.AddCommand(initDataCmd)
}
