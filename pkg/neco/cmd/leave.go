package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var leaveParams struct {
	lrn int
}

// leaveCmd represents the leave command
var leaveCmd = &cobra.Command{
	Use:   "leave LRN",
	Short: "Unregister LRN of the boot server from etcd.",
	Long:  `Unregister LRN of the boot server from etcd.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		num, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			return err
		}
		leaveParams.lrn = int(num)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("leave %d from cluster\n", leaveParams.lrn)
	},
}

func init() {
	rootCmd.AddCommand(leaveCmd)
}
