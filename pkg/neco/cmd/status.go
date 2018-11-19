package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

func showStatus(ctx context.Context, st storage.Storage, w io.Writer) error {
	ss, err := st.NewSnapshot(ctx)
	if err != nil {
		return err
	}

	req := ss.Request
	lrns := ss.Servers
	statuses := ss.Statuses

	fmt.Fprintln(w, "Boot servers:", lrns)
	fmt.Fprintln(w, "Update process")
	if req == nil {
		fmt.Fprintln(w, "    status: clear")
		return nil
	}

	switch {
	case checkUpdateAborted(req.Version, statuses):
		fmt.Fprintln(w, "    status: aborted")
	case neco.UpdateCompleted(req.Version, req.Servers, statuses):
		fmt.Fprintln(w, "    status: completed")
	default:
		fmt.Fprintln(w, "    status: running")
	}

	fmt.Fprintln(w, "   version:", req.Version)
	fmt.Fprintln(w, "   members:", req.Servers)
	fmt.Fprintln(w, "   started:", req.StartedAt.Format(time.RFC3339))

	if len(statuses) == 0 {
		return nil
	}

	bs := make([]int, 0, len(statuses))
	for k := range statuses {
		bs = append(bs, k)
	}
	sort.Ints(bs)

	for _, lrn := range bs {
		status := statuses[lrn]
		if status.Version != req.Version {
			continue
		}
		fmt.Fprintf(w, "\nBoot server %d\n", lrn)
		fmt.Fprintln(w, "    step:", status.Step)
		fmt.Fprintln(w, "    condition:", status.Cond.String())
		if len(status.Message) > 0 {
			fmt.Fprintln(w, "    message:", status.Message)
		}
	}

	return nil
}
func checkUpdateAborted(ver string, statuses map[int]*neco.UpdateStatus) bool {
	for _, status := range statuses {
		if status.Version != ver {
			continue
		}
		if status.Cond == neco.CondAbort {
			return true
		}
	}
	return false
}

// statusCmd implements "neco status"
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "show the update process status",
	Long:  `Show the status of the current update process.`,
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			return showStatus(ctx, st, os.Stdout)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
