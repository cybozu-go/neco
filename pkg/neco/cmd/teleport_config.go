package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"

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

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return teleportConfig(context.Background())
	},
}

func init() {
	teleportCmd.AddCommand(teleportConfigCmd)
}

func teleportConfig(ctx context.Context) error {
	if os.Getuid() != 0 {
		return errors.New("run as root")
	}

	ce, err := neco.EtcdClient()
	if err != nil {
		return err
	}
	defer ce.Close()
	st := storage.NewStorage(ce)

	token, err := st.GetTeleportAuthToken(ctx)
	if err != nil {
		return err
	}
	if len(token) == 0 {
		return errors.New("no teleport token is found")
	}

	authServers, err := st.GetTeleportAuthServers(context.Background())
	if err != nil {
		return err
	}

	mylrn, err := neco.MyLRN()
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	err = teleport.GenerateConf(buf, mylrn, string(token), authServers)
	if err != nil {
		return err
	}
	err = os.WriteFile(neco.TeleportConfFile, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	return neco.RestartService(context.Background(), "teleport-node")
}
