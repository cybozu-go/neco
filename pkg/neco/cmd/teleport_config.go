package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var teleportConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "generate teleport config",
	Long: `Generate config for teleport by filling template with secret in file
and dynamic info in etcd.`,

	Run: func(cmd *cobra.Command, args []string) {
		if os.Getuid() != 0 {
			log.ErrorExit(errors.New("run as root"))
		}

		token, err := ioutil.ReadFile(neco.TeleportTokenFile)
		if err != nil {
			log.ErrorExit(err)
		}
		confBase, err := ioutil.ReadFile(neco.TeleportConfFileBase)
		if err != nil {
			log.ErrorExit(err)
		}

		ce, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ce.Close()
		st := storage.NewStorage(ce)

		authServers, err := getAuthServers(st)
		if err != nil {
			log.ErrorExit(err)
		}

		conf, err := generateConfig(token, authServers, confBase)
		if err != nil {
			log.ErrorExit(err)
		}
		err = ioutil.WriteFile(neco.TeleportConfFile, conf, 0600)
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	teleportCmd.AddCommand(teleportConfigCmd)
}

func generateConfig(token []byte, authServers []string, base []byte) ([]byte, error) {
	authServersJSON, err := json.Marshal(authServers)
	if err != nil {
		return nil, err
	}

	conf := bytes.ReplaceAll(base, []byte("%AUTH_TOKEN%"), token)
	conf = bytes.ReplaceAll(conf, []byte("%AUTH_SERVERS%"), authServersJSON)

	return conf, nil
}

func getAuthServers(st storage.Storage) ([]string, error) {
	var authServers []string
	well.Go(func(ctx context.Context) error {
		var err error
		authServers, err = st.GetTeleportAuthServers(ctx)
		if err != nil && err != storage.ErrNotFound {
			log.Warn("unexpected error in getting auth servers info", map[string]interface{}{
				log.FnError: err,
			})
		}
		return err
	})
	well.Stop()
	err := well.Wait()
	if err != nil {
		return nil, err
	}

	return authServers, nil
}
