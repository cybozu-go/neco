package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/netutil"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

const placematParamDevice = "/dev/virtio-ports/placemat"
const placematBMCAddressBase = "10.72.17.0"

var rootParams struct {
	checkOnly bool
	name      string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "setup-hw",
	Short: "Hardware settings updater/checker",
	Long: `setup-hw is a command-line tool for managing hardware settings.
It updates hardware settings as expected by default.
Hardware name can be specified with a "--name" option.
If it is called with a "--check-only" option, it just checks settings.

Not all flags are supported by all hardware types.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := well.LogConfig{}.Apply()
		if err != nil {
			log.ErrorExit(err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		hw, err := neco.DetectHardware()
		if err != nil {
			log.ErrorExit(err)
		}
		switch hw {
		case neco.HWTypeVM:
			err = rootPlacematVM()
		case neco.HWTypeDell:
			//err = rootDell()
		default:
			err = fmt.Errorf("unknown hardware type: %v", hw)
		}
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&rootParams.checkOnly, "check-only", "c", false, "check settings but do not update")
	rootCmd.Flags().StringVar(&rootParams.name, "name", "", "set hardware name")
	rootCmd.Flags().StringVar(&rootParams.name, "rac-name", "", "set hardware name (deprecated)")
}

func rootPlacematVM() error {
	if rootParams.checkOnly {
		return errors.New("--check-only is not supported for Placemat VM")
	}
	if len(rootParams.name) != 0 {
		return errors.New("--name is not supported for Placemat VM")
	}

	lrn, err := neco.MyLRN()
	if err != nil {
		return err
	}

	base := netutil.IP4ToInt(net.ParseIP(placematBMCAddressBase))
	addr := netutil.IntToIP4(base + 32*uint32(lrn) + 3)

	f, err := os.OpenFile(placematParamDevice, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	f.WriteString(addr.String() + "\n")
	f.Close()
	return nil
}

func rootDell() error {
	command := []string{"setup-hw"}
	if rootParams.checkOnly {
		command = append(command, "--check-only")
	}
	if len(rootParams.name) != 0 {
		command = append(command, "--rac-name", rootParams.name)
	}

	cmd, err := neco.EnterContainerAppCommand(context.Background(), "omsa", command)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
