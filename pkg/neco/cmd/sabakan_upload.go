package cmd

import (
	"context"

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

func sabakanUpload(ctx context.Context, st storage.Storage) error {
	version, err := neco.GetDebianVersion(neco.NecoPackageName)
	if err != nil {
		return err
	}

	proxyClient, err := ext.ProxyHTTPClient(ctx, st)
	if err != nil {
		return err
	}
	localClient := ext.LocalHTTPClient()

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
	env.Stop()
	return env.Wait()
}

// sabakanUploadCmd implements "sabakan-upload"
var sabakanUploadCmd = &cobra.Command{
	Use:   "sabakan-upload",
	Short: "Upload sabakan contents using artifacts.go",
	Long: `Upload sabakan contents using artifacts.go
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
			return sabakanUpload(ctx, st)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	sabakanUploadCmd.Flags().BoolVar(&ignitionsOnly, "ignitions-only", false, "upload ignitions only")
	rootCmd.AddCommand(sabakanUploadCmd)
}
