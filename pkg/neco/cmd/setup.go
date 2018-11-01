package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var setupParams struct {
	lrns []int
	noRevoke bool
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup  [--no-revoke] LRN [LRN ...]",
	Short: "Install and setup etcd cluster as well as Vault using given boot servers",
	Long: `Install and setup etcd cluster as well as Vault using given boot
servers. LRN is the logical rack number of the boot server. At least 3
LRNs should be specified.

This command should be invoked at once on all boot servers specified by LRN.

When --no-revoke option is specified, it does not remove the etcd key
<prefix>/vault-root-token. This option is used by automatic setup of dctest`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return errors.New("too few boot servers")
		}
		setupParams.lrns = make([]int, len(args))
		for i, a := range args {
			num, err := strconv.ParseUint(a, 10, 32)
			if err != nil {
				return err
			}
			setupParams.lrns[i] = int(num)
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("setup boot servers with lrn: ", setupParams.lrns)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().BoolVar(&setupParams.noRevoke, "no-revoke", false, "keep vault root token in etcd")
}
