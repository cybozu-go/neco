package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var bmcRepairCmd = &cobra.Command{
	Use:   "repair BMC_TYPE BMC_specific_command...",
	Short: "repair a machine via BMC",
	Long:  `Try to repair an unhealthy/unreachable machine by invoking BMC functions remotely.`,
}

func init() {
	bmcCmd.AddCommand(bmcRepairCmd)
}

func getBMCWithType(ctx context.Context, id, bmcType string) (*sabakan.MachineBMC, error) {
	machine, err := lookupMachine(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup serial or IP address %q: %w", id, err)
	}

	if machine.Spec.BMC.Type != bmcType {
		return nil, fmt.Errorf("not a machine with %q-type BMC: %q", bmcType, id)
	}

	return &machine.Spec.BMC, nil
}

func dialToBMCByRepairUser(ctx context.Context, bmc *sabakan.MachineBMC) (*ssh.Client, error) {
	var address string
	switch {
	case len(bmc.IPv6) > 0:
		address = bmc.IPv6
	case len(bmc.IPv4) > 0:
		address = bmc.IPv4
	default:
		return nil, errors.New("BMC IP address not set")
	}

	etcd, err := neco.EtcdClient()
	if err != nil {
		return nil, err
	}
	defer etcd.Close()

	st := storage.NewStorage(etcd)
	user, err := st.GetBMCRepairUser(ctx)
	if err != nil {
		return nil, err
	}
	password, err := st.GetBMCRepairPassword(ctx)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) (answers []string, err error) {
				ret := make([]string, len(questions))
				for i := range questions {
					ret[i] = password
				}
				return ret, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // acceptable for local network
	}

	return ssh.Dial("tcp", address+":22", config)
}

func sshSessionOutput(client *ssh.Client, cmd string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.Stderr = os.Stderr
	return session.Output(cmd)
}
