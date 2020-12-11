package cmd

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/teleport"
	"github.com/cybozu-go/neco/storage"
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

		ce, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer ce.Close()
		st := storage.NewStorage(ce)

		authServers, err := st.GetTeleportAuthServers(context.Background())
		if err != nil {
			log.ErrorExit(err)
		}

		mylrn, err := neco.MyLRN()
		if err != nil {
			log.ErrorExit(err)
		}

		buf := &bytes.Buffer{}
		err = teleport.GenerateConf(buf, mylrn, string(token), authServers)
		if err != nil {
			log.ErrorExit(err)
		}
		err = ioutil.WriteFile(neco.TeleportConfFile, buf.Bytes(), 0600)
		if err != nil {
			log.ErrorExit(err)
		}

		err = neco.RestartService(context.Background(), "teleport-node")
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	teleportCmd.AddCommand(teleportConfigCmd)
}
