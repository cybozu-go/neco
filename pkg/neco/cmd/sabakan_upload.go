package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var sabakanUploadParams = struct {
	quayUser     string
	quayPassword string
}{
	quayUser: "cybozu+neco_readonly",
}

// sabakanUploadCmd implements "sabakan-upload"
var sabakanUploadCmd = &cobra.Command{
	Use:   "sabakan-upload",
	Short: "Upload sabakan contents using artifacts.go",
	Long: `Upload sabakan contents using artifacts.go
If uploaded versions are up to date, do nothing.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		user := os.Getenv("QUAY_USER")
		if len(user) > 0 {
			sabakanUploadParams.quayUser = user
		}

		passwd := os.Getenv("QUAY_PASSWORD")
		if len(passwd) == 0 {
			return errors.New("QUAY_PASSWORD envvar is not set")
		}
		sabakanUploadParams.quayPassword = passwd

		return nil
	},

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

			auth := neco.DockerAuth{
				Username: sabakanUploadParams.quayUser,
				Password: sabakanUploadParams.quayPassword,
			}

			return sabakan.UploadContents(ctx, localClient, proxyClient, version, auth)
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
