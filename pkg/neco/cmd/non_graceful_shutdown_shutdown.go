package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nonGracefulNodeShutdownShutdownCmd = &cobra.Command{
	Use:   "shutdown IP_ADDRESS",
	Short: "non-graceful shutdown the node",
	Long:  `Power off the node and create NetworkFence and then add taint to the node`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node := args[0]

		ctx := context.Background()
		kubeClient, err := issueAndLoadKubeconfigForNonGracefulNodeShutdown()
		if err != nil {
			return err
		}

		kubernetesNode := &corev1.Node{}
		err = kubeClient.Get(ctx, client.ObjectKey{Name: node}, kubernetesNode)
		if err != nil {
			if apierrors.IsNotFound(err) {
				fmt.Printf("Node %s is not k8s node, skipping\n", node)
				return nil
			}
			return err
		}

		fmt.Printf("Shutting down the node: %s\n", node)
		out, err := exec.Command("neco", "power", "status", node).Output()
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(out)) == "On" {
			err = exec.Command("neco", "power", "stop", node).Run()
			if err != nil {
				return err
			}
			fmt.Printf("Waiting for the node %s to be power off\n", node)
			for {
				out, err := exec.Command("neco", "power", "status", node).Output()
				if err != nil {
					return err
				}
				if strings.TrimSpace(string(out)) == "Off" {
					break
				}
				time.Sleep(5 * time.Second)
			}
		}
		fmt.Printf("Node %s is power off\n", node)

		g := errgroup.Group{}
		cephClusters, err := listRBDCephClusters(ctx, kubeClient)
		if err != nil {
			return err
		}
		for _, cephCluster := range cephClusters {
			cephCluster := cephCluster
			g.Go(func() error {
				fenceName := generateFenceName(cephCluster.Name, node)
				networkFence := csiaddonsv1alpha1.NetworkFence{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fenceName,
						Namespace: cephCluster.Namespace,
					},
					Spec: csiaddonsv1alpha1.NetworkFenceSpec{
						FenceState: csiaddonsv1alpha1.Fenced,
						Driver:     cephCluster.Name + ".rbd.csi.ceph.com",
						Cidrs:      []string{node + "/32"},
						Secret: csiaddonsv1alpha1.SecretSpec{
							Name:      "rook-csi-rbd-provisioner",
							Namespace: cephCluster.Namespace,
						},
						Parameters: map[string]string{
							"clusterID": cephCluster.Namespace,
						},
					},
				}
				err = kubeClient.Create(ctx, &networkFence)
				if err != nil {
					if apierrors.IsAlreadyExists(err) {
						fmt.Printf("NetworkFence %s already exists\n", networkFence.Name)
					} else {
						return err
					}
				}
				fmt.Printf("Waiting for the fence operation of %s to be succeeded\n", networkFence.Name)
				networkFence = csiaddonsv1alpha1.NetworkFence{}
				for {
					err := kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, &networkFence)
					if err != nil {
						return err
					}
					if networkFence.Status.Result == csiaddonsv1alpha1.FencingOperationResultSucceeded {
						break
					}
					time.Sleep(5 * time.Second)
				}
				fmt.Printf("Fence operation for NetworkFence %s is succeeded\n", networkFence.Name)
				return nil
			})
		}
		err = g.Wait()
		if err != nil {
			return err
		}

		// Add taint to the node
		fmt.Println("Adding taint to the node")
		err = kubeClient.Get(ctx, client.ObjectKey{Name: node}, kubernetesNode)
		if err != nil {
			return err
		}
		tainted := false
		for _, taint := range kubernetesNode.Spec.Taints {
			if taint.Key == corev1.TaintNodeOutOfService {
				tainted = true
				break
			}
		}
		if !tainted {
			kubernetesNode.Spec.Taints = append(kubernetesNode.Spec.Taints, corev1.Taint{
				Key:    corev1.TaintNodeOutOfService,
				Value:  "nodeshutdown",
				Effect: corev1.TaintEffectNoExecute,
			})
			err = kubeClient.Update(ctx, kubernetesNode)
			if err != nil {
				return err
			}
		}

		fmt.Println("Waiting for the VolumeAttachment to be deleted")
		for {
			volumeAttachmentList := &storagev1.VolumeAttachmentList{}
			err = kubeClient.List(ctx, volumeAttachmentList)
			if err != nil {
				return err
			}
			volumeAttachmenCount := 0
			for _, volumeAttachment := range volumeAttachmentList.Items {
				if volumeAttachment.Spec.NodeName == node {
					volumeAttachmenCount++
				}
			}
			if volumeAttachmenCount == 0 {
				break
			}
			time.Sleep(5 * time.Second)
		}
		fmt.Println("Non-Graceful Node Shutdown completed")
		return nil
	},
}

func init() {
	nonGracefulNodeShutdownCmd.AddCommand(nonGracefulNodeShutdownShutdownCmd)
}
