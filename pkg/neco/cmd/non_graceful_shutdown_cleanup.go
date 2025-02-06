package cmd

import (
	"context"
	"fmt"
	"slices"
	"time"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nonGracefulShutdownCleanupCmd = &cobra.Command{
	Use:   "cleanup IP_ADDRESS",
	Short: "Cleanup non-graceful shutdowned node",
	Long:  `Remove NetworkFence and remove taint from the node if it is healthy`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node := args[0]

		ctx := context.Background()

		kubeClient, err := issueAndLoadKubeconfigForNonGracefulNodeShutdown()
		if err != nil {
			return err
		}

		opt := sabakanMachinesGetOpts{}
		opt.params = map[string]*string{
			"ipv4": &node,
		}
		machines, err := sabakanMachinesGet(ctx, &opt)
		if err != nil {
			return err
		}
		sabakanStatus := machines[0].Status.State

		cephClusters, err := listRBDCephClusters(ctx, kubeClient)
		if err != nil {
			return err
		}
		g := errgroup.Group{}
		for _, cephCluster := range cephClusters {
			cephCluster := cephCluster
			g.Go(func() error {
				fenceName := generateFenceName(cephCluster.Name, node)
				networkFence := &csiaddonsv1alpha1.NetworkFence{}
				err = kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, networkFence)
				if err != nil {
					if client.IgnoreNotFound(err) == nil {
						fmt.Println("NetworkFence is already removed")
						return nil
					} else {
						return err
					}
				}
				networkFence.Spec.FenceState = csiaddonsv1alpha1.Unfenced
				err = kubeClient.Update(ctx, networkFence)
				if err != nil {
					return err
				}
				fmt.Printf("Waiting for Unfence operation of %s to be Succeeded\n", networkFence.Name)
				for {
					err := kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, networkFence)
					if err != nil {
						return err
					}
					if networkFence.Status.Result == csiaddonsv1alpha1.FencingOperationResultSucceeded && networkFence.Status.Message == csiaddonsv1alpha1.UnFenceOperationSuccessfulMessage {
						break
					}
					time.Sleep(5 * time.Second)
				}
				err = kubeClient.Delete(ctx, networkFence)
				if err != nil {
					return err
				}
				fmt.Printf("Unfence operation for NetworkFence %s is succeeded and it is removed\n", networkFence.Name)
				return nil
			})
		}
		err = g.Wait()
		if err != nil {
			return err
		}

		if sabakanStatus == sabakan.StateHealthy {
			kubernetesNode := &corev1.Node{}
			err = kubeClient.Get(ctx, client.ObjectKey{Name: node}, kubernetesNode)
			if err != nil {
				return err
			}
			for i, taint := range kubernetesNode.Spec.Taints {
				if taint.Key == outOfServiceTaintKey {
					kubernetesNode.Spec.Taints = slices.Delete(kubernetesNode.Spec.Taints, i, i+1)
				}
			}
			err = kubeClient.Update(ctx, kubernetesNode)
			if err != nil {
				return err
			}
			fmt.Println("Removed taint from the node")
		}
		fmt.Println("Non-Graceful Node Shutdown cleanup completed")
		return nil
	},
}

func init() {
	nonGracefulNodeShutdownCmd.AddCommand(nonGracefulShutdownCleanupCmd)
}
