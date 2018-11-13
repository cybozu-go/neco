package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var joinParams struct {
	lrns []int
}

// joinCmd represents the join command
var joinCmd = &cobra.Command{
	Use:   "join LRN [LRN ...]",
	Short: "Prepare this server to join the cluster.",
	Long: `Prepare certificates and files to add this server to the cluster.

LRN are a list of LRNs of the existing boot servers.

To issue certificates, this command asks the user Vault username and password.
It also creates "/etc/neco/config.yml" for neco-updater and neco-worker.

Etcd and Vault themselves are *not* installed by this command.  They are
installed later by neco-worker.  Similarly, this command does not
add the new server to etcd cluster.  neco-worker will add the server
to etcd cluster.`,

	Args: func(cmd *cobra.Command, args []string) error {
		joinParams.lrns = make([]int, len(args))
		for i, a := range args {
			num, err := strconv.ParseUint(a, 10, 32)
			if err != nil {
				return err
			}
			joinParams.lrns[i] = int(num)
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("join to cluster %v\n", joinParams.lrns)
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}
