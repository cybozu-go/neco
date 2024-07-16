package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var rebootListListOptions struct {
	Output string
}

var rebooterListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the entries in the reboot list",
	Long:  `List the entries in the reboot list.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		entries, err := necoStorage.GetRebootListEntries(ctx)
		if err != nil {
			return err
		}
		if rebootListListOptions.Output == "simple" {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 1, 1, ' ', 0)
			w.Write([]byte("Index\tNode\tGroup\tRebootTime\tStatus\n"))
			for _, entry := range entries {
				w.Write([]byte(fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t\n", entry.Index, entry.Node, entry.Group, entry.RebootTime, entry.Status)))
			}
			return w.Flush()
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			if err := enc.Encode(entries); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rebooterListCmd.Flags().StringVarP(&rebootListListOptions.Output, "output", "o", "json", "Output format [json,simple]")
	rebooterCmd.AddCommand(rebooterListCmd)
}
