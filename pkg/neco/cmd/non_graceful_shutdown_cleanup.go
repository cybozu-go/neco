package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nonGracefulShutdownCleanupCmd = &cobra.Command{
	Use:   "cleanup IP_ADDRESS",
	Short: "Cleanup non-graceful shutdowned node",
	Long:  `Remove NetworkFence and remove taint from the node if it is healthy`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node := args[0]

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		kubeClient, err := IssueAndLoadKubeconfigForNonGracefulNodeShutdown()
		if err != nil {
			return err
		}

		//get sabakan status
		opt := sabakanMachinesGetOpts{}
		opt.params = map[string]*string{
			"ipv4": &node,
		}
		machines, err := sabakanMachinesGet(ctx, &opt)
		if err != nil {
			return err
		}
		sabakanStatus := machines[0].Status.State

		// remove networkfence
		g := errgroup.Group{}
		for _, cephClusterName := range cephClusters {
			cephClusterName := cephClusterName
			//check cephCluster exists
			cephCluster := &unstructured.Unstructured{}
			cephCluster.SetAPIVersion("ceph.rook.io/v1")
			cephCluster.SetKind("CephCluster")
			err := kubeClient.Get(ctx, client.ObjectKey{Name: cephClusterName, Namespace: cephClusterName}, cephCluster)
			if err != nil {
				if client.IgnoreNotFound(err) == nil {
					fmt.Printf("CephCluster %s does not found\n", cephClusterName)
					continue
				} else {
					return err
				}
			}
			g.Go(func() error {
				fenceName := cephClusterName + "-" + strings.Replace(node, ".", "-", -1)
				networkFence := &csiaddonsv1alpha1.NetworkFence{}
				err = kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, networkFence)
				if err != nil {
					if client.IgnoreNotFound(err) == nil {
						fmt.Printf("NetworkFence %s already removed\n", networkFence.Name)
						return nil
					} else {
						return err
					}
				}
				networkFence.Spec.FenceState = csiaddonsv1alpha1.Unfenced
				networkFence.Status = csiaddonsv1alpha1.NetworkFenceStatus{}
				err = kubeClient.Update(ctx, networkFence)
				if err != nil {
					return err
				}

				// wait for unfense of networkfence to be Succeeded
				timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				for {
					select {
					case <-timeoutCtx.Done():
						return errors.New("timeout waiting for networkfence to be unfenced")
					default:
					}
					err := kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, networkFence)
					if err != nil {
						return err
					}
					if networkFence.Status.Result == csiaddonsv1alpha1.FencingOperationResultSucceeded {
						break
					}
					time.Sleep(5 * time.Second)
				}
				err = kubeClient.Delete(ctx, networkFence)
				if err != nil {
					return err
				}
				return nil
			})
		}

		if sabakanStatus == sabakan.StateHealthy {
			kubernetesNode := &corev1.Node{}
			err = kubeClient.Get(ctx, client.ObjectKey{Name: node}, kubernetesNode)
			if err != nil {
				return err
			}
			for i, taint := range kubernetesNode.Spec.Taints {
				if taint.Key == "node.kubernetes.io/out-of-service" {
					kubernetesNode.Spec.Taints = slices.Delete(kubernetesNode.Spec.Taints, i, i+1)
				}
			}
			err = kubeClient.Update(ctx, kubernetesNode)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	nonGracefulNodeShutdownCmd.AddCommand(nonGracefulShutdownCleanupCmd)
}
