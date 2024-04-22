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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var addCmd = &cobra.Command{
	Use:   "add FILE",
	Short: "append the nodes written in FILE to the reboot list",
	Long:  `Append the nodes written in FILE to the reboot list.`,
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

	OUTER:
		for _, node := range nodes {
			kubeNode, err := validateNode(node, *kubernetesNodes)
			if err != nil {
				return err
			}
			group, ok := kubeNode.ObjectMeta.Labels[config.GroupLabelKey]
			if !ok {
				return fmt.Errorf("node has no %s label", config.GroupLabelKey)
			}
			for _, rt := range config.RebootTimes {
				matches := make([]bool, 0)
				for key, value := range rt.LabelSelector.MatchLabels {
					if kubeNode.ObjectMeta.Labels[key] == value {
						matches = append(matches, true)
					} else {
						matches = append(matches, false)
					}
				}
				if all(matches) {
					fmt.Printf("adding a node to RebootListEntry node=%s  group=%s reboot_time=%s\n", kubeNode.Name, group, rt.Name)
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
					continue OUTER
				}
			}
		}
		return nil
	},
}

func validateNode(node string, kubeNode v1.NodeList) (*v1.Node, error) {
	for _, n := range kubeNode.Items {
		if n.Name == node {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("%s is not a valid node IP address", node)
}

func init() {
	rebooterCmd.AddCommand(addCmd)
}
