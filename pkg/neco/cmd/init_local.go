package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

var initLocalParams struct {
	name string
}

// initLocalCmd represents the initLocal command
var initLocalCmd = &cobra.Command{
	Use:   "init-local NAME",
	Short: "Initialize data for new application of a boot server executes",
	Long: `Initialize data for new application of a boot server executes. This
command should not be executed more than once.  It asks vault user and
password to generate a vault token, then issue client certificates for
new a application NAME.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		initLocalParams.name = args[0]
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		mylrn, err := neco.MyLRN()
		if err != nil {
			log.ErrorExit(err)
		}
		vc, err := neco.VaultClient(mylrn)
		if err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			var err error
			switch initLocalParams.name {
			case "etcdpasswd":
				err = issueCerts(ctx, vc, "etcdpasswd", neco.EtcdpasswdCertFile, neco.EtcdpasswdKeyFile)
			case "teleport":
				err = getToken(ctx, vc, "teleport", neco.TeleportTokenFile)
			case "sabakan":
				err = issueCerts(ctx, vc, "sabakan", neco.SabakanCertFile, neco.SabakanKeyFile)
				if err != nil {
					return err
				}
				err = sabakan.InitLocal(ctx, vc)
			case "cke":
				err = issueCerts(ctx, vc, "cke", neco.CKECertFile, neco.CKEKeyFile)
			default:
				return errors.New("unknown service name: " + initLocalParams.name)
			}
			if err != nil {
				return err
			}

			switch initLocalParams.name {
			case "etcdpasswd":
				err = neco.StartService(ctx, neco.EtcdpasswdService)
			case "teleport":
				err = neco.StartService(ctx, neco.TeleportService)
			case "sabakan":
				err = neco.StartService(ctx, neco.SabakanService)
			case "cke":
				err = neco.StartService(ctx, neco.CKEService)
			}
			return err
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initLocalCmd)
}

func issueCerts(ctx context.Context, vc *api.Client, commonName, cert, key string) error {
	secret, err := vc.Logical().Write(neco.CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          commonName,
		"exclude_cn_from_sans": true,
	})
	if err != nil {
		return err
	}
	err = neco.WriteFile(cert, secret.Data["certificate"].(string))
	if err != nil {
		return err
	}
	return neco.WriteFile(key, secret.Data["private_key"].(string))

}

func getToken(ctx context.Context, vc *api.Client, path string, filename string) error {
	secret, err := vc.Logical().Read(fmt.Sprintf("secret/%s", path))
	if err != nil {
		return err
	}
	if secret == nil || len(secret.Data) == 0 {
		return fmt.Errorf("reading secret/%s returned nil or empty secret", path)
	}

	token, ok := secret.Data["token"]
	if !ok || len(token) == 0 {
		return fmt.Errorf("secret/%s does not contain token field, or contains but empty", path)
	}

	token_string, ok := token.(string)
	if !ok {
		return fmt.Errorf("secret/%s contains non-string token field", path)
	}

	err = ioutil.WriteFile(filename, []byte(token_string), 0600)
	return err
}
