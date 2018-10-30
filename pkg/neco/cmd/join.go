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
	Short: "Join this server as a new boot server and an etcd member.",
	Long: `Join this server as a new boot server and an etcd member. LRN is the logical
rack number of current available boot servers. At least 3 LRNs should be
specified.  It asks vault user and password to generate a vault token, then
issue client certificates for etcd and vault for a new boot server.`,

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
