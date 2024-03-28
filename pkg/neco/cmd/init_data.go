package cmd

import (
	"context"
	"net/http"
	"net/url"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/progs/cke"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/spf13/cobra"
)

var ignitionsOnly, updateResourcesOnly, uploadCACertOnly bool

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

	proxy, err := st.GetProxyConfig(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	transport := http.DefaultTransport
	if len(proxy) != 0 {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return err
		}

		t := http.DefaultTransport.(*http.Transport).Clone()
		t.Proxy = http.ProxyURL(proxyURL)
		transport = t
	}

	username, err := st.GetGhcrUsername(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	password, err := st.GetGhcrPassword(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	var auth authn.Authenticator
	if len(username) != 0 && len(password) != 0 {
		auth = &authn.Basic{
			Username: username,
			Password: password,
		}
	}

	fetcher := neco.NewImageFetcher(transport, auth)

	if ignitionsOnly {
		return sabakan.UploadIgnitions(ctx, localClient, version, st)
	}

	if updateResourcesOnly {
		return cke.UpdateResources(ctx, st)
	}

	if uploadCACertOnly {
		return sabakan.UploadCACert(ctx, localClient)
	}

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return sabakan.UploadContents(ctx, localClient, proxyClient, version, fetcher, st)
	})
	env.Go(func(ctx context.Context) error {
		return cke.UploadContents(ctx, localClient, proxyClient, version, fetcher)
	})
	env.Go(func(ctx context.Context) error {
		return sabakan.UploadDHCPJSON(ctx, localClient)
	})
	env.Go(func(ctx context.Context) error {
		return cke.SetCKETemplate(ctx, st)
	})
	env.Go(func(ctx context.Context) error {
		return cke.UpdateResources(ctx, st)
	})
	env.Go(func(ctx context.Context) error {
		return sabakan.UploadCACert(ctx, localClient)
	})
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
	initDataCmd.Flags().BoolVar(&updateResourcesOnly, "update-resources-only", false, "update user-defined resources only")
	initDataCmd.Flags().BoolVar(&uploadCACertOnly, "upload-ca-cert-only", false, "upload ca certificate of sabakan to assets only")

	rootCmd.AddCommand(initDataCmd)
}
