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

// sabakanUploadCmd implements "sabakan-upload"
var sabakanUploadCmd = &cobra.Command{
	Use:   "sabakan-upload",
	Short: "Upload sabakan contents using artifacts.go",
	Long: `Upload sabakan contents using artifacts.go
If uploaded versions are up to date, do nothing.
`,
	Run: func(cmd *cobra.Command, args []string) {
		version, err := neco.GetDebianVersion(neco.NecoPackageName)
		if err != nil {
			log.ErrorExit(err)
		}

		ec, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ec.Close()
		st := storage.NewStorage(ec)

		well.Go(func(ctx context.Context) error {
			proxyClient, err := ext.ProxyHTTPClient(ctx, st)
			if err != nil {
				return err
			}
			localClient := ext.LocalHTTPClient()

			username, err := st.GetQuayUsername(ctx)
			if err != nil {
				return err
			}
			password, err := st.GetQuayPassword(ctx)
			if err != nil {
				return err
			}

			auth := &neco.DockerAuth{
				Username: username,
				Password: password,
			}

			err = sabakan.UploadContents(ctx, localClient, proxyClient, version, auth, st)
			if err != nil {
				return err
			}

			return cke.UploadContents(ctx, localClient, proxyClient, version)
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(sabakanUploadCmd)
}
