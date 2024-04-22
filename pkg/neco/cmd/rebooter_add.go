package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rebooterAddCmd = &cobra.Command{
	Use:   "add FILE",
	Short: "append the nodes written in FILE to the reboot list",
	Long:  `Append the nodes written in FILE to the reboot list.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		f := os.Stdin
		if args[0] != "-" {
			var err error
			f, err = os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
		}

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		nodes := strings.Fields(string(data))

		retryCount := 0
	RETRY:
		kubernetesNodes, err := KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			if retryCount > 2 {
				return err
			}
			err := renewKubeConfig()
			if err != nil {
				return err
			}
			retryCount++
			goto RETRY
		}

		for _, node := range nodes {
			kubeNode, err := validateNode(node, *kubernetesNodes)
			if err != nil {
				return err
			}
			group, ok := kubeNode.ObjectMeta.Labels[config.GroupLabelKey]
			if !ok {
				return fmt.Errorf("node has no groupKey label (%s)", config.GroupLabelKey)
			}
			rt, err := matchRebootTimes(*kubeNode, config.RebootTimes)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("adding a node to reboot list node=%s  group=%s reboot_time=%s\n", kubeNode.Name, group, rt.Name)
			newEntry := neco.RebootListEntry{
				Node:       kubeNode.Name,
				Group:      group,
				RebootTime: rt.Name,
				Status:     neco.RebootListEntryStatusPending,
			}
			if !flagDryRun {
				err := necoStorage.RegisterRebootListEntry(ctx, &newEntry)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func validateNode(node string, kubeNode corev1.NodeList) (*corev1.Node, error) {
	for _, n := range kubeNode.Items {
		if n.Name == node {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("%s is not a valid node IP address", node)
}

func init() {
	rebooterAddCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "dry-run")
	rebooterCmd.AddCommand(rebooterAddCmd)
}
