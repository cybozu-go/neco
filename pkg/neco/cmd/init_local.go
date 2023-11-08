package cmd

import (
	"context"
	"errors"
	"os"

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
		ce, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ce.Close()

		well.Go(func(ctx context.Context) error {
			switch initLocalParams.name {
			case "etcdpasswd":
				err := issueEtcdCerts(ctx, vc, "etcdpasswd", neco.EtcdpasswdCertFile, neco.EtcdpasswdKeyFile)
				if err != nil {
					if err != os.ErrExist {
						return err
					}
					log.Info("certificate already exists, skipping issue cert", map[string]interface{}{
						"cert": neco.EtcdpasswdCertFile,
					})
				}
			case "sabakan":
				err := issueEtcdCerts(ctx, vc, "sabakan", neco.SabakanEtcdCertFile, neco.SabakanEtcdKeyFile)
				if err != nil {
					if err != os.ErrExist {
						return err
					}
					log.Info("certificate already exists, skipping issue cert", map[string]interface{}{
						"cert": neco.SabakanEtcdCertFile,
					})
				}
				err = issueSabakanServerCerts(ctx, vc, neco.SabakanServerCertFile, neco.SabakanServerKeyFile)
				if err != nil {
					if err != os.ErrExist {
						return err
					}
					log.Info("certificate already exists, skipping issue cert", map[string]interface{}{
						"cert": neco.SabakanServerCertFile,
					})
				}
				err = sabakan.InitLocal(ctx, vc)
				if err != nil {
					return err
				}
			case "cke":
				err := issueEtcdCerts(ctx, vc, "cke", neco.CKECertFile, neco.CKEKeyFile)
				if err != nil {
					if err != os.ErrExist {
						return err
					}
					log.Info("certificate already exists, skipping issue cert", map[string]interface{}{
						"cert": neco.CKECertFile,
					})
				}
			default:
				return errors.New("unknown service name: " + initLocalParams.name)
			}

			switch initLocalParams.name {
			case "etcdpasswd":
				isActive, err := neco.IsActiveService(ctx, neco.EtcdpasswdService)
				if err != nil {
					return err
				}
				if !isActive {
					return neco.StartService(ctx, neco.EtcdpasswdService)
				}
			case "sabakan":
				isActive, err := neco.IsActiveService(ctx, neco.SabakanService)
				if err != nil {
					return err
				}
				if !isActive {
					return neco.StartService(ctx, neco.SabakanService)
				}
			case "cke":
				isActive, err := neco.IsActiveService(ctx, neco.CKEService)
				if err != nil {
					return err
				}
				if !isActive {
					err := neco.StartService(ctx, neco.CKEService)
					if err != nil {
						return err
					}
					return neco.StartService(ctx, neco.CKELocalProxyService)
				}
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
	rootCmd.AddCommand(initLocalCmd)
}

func issueEtcdCerts(ctx context.Context, vc *api.Client, commonName, cert, key string) error {
	_, err := os.Stat(cert)
	if err == nil {
		return os.ErrExist
	}
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

func issueSabakanServerCerts(ctx context.Context, vc *api.Client, cert, key string) error {
	_, err := os.Stat(cert)
	if err == nil {
		return os.ErrExist
	}
	myname, err := os.Hostname()
	if err != nil {
		return err
	}
	rack, err := neco.MyLRN()
	if err != nil {
		return err
	}
	bootServerAddress := neco.BootNode0IP(rack).String()
	secret, err := vc.Logical().Write(neco.CAServer+"/issue/system", map[string]interface{}{
		"common_name": myname,
		"alt_names":   "localhost",
		"ip_sans":     []string{"127.0.0.1", bootServerAddress},
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
