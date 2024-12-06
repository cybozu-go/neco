package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nonGracefulNodeShutdownShutdownCmd = &cobra.Command{
	Use:   "shutdown IP_ADDRESS",
	Short: "non-graceful shutdown the node",
	Long:  `power off the node and create NetworkFence and then add taint to the node`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node := args[0]

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		kubeClient, err := IssueAndLoadKubeconfigForNonGracefulNodeShutdown()
		if err != nil {
			return err
		}

		kubernetesNode := &corev1.Node{}
		err = kubeClient.Get(ctx, client.ObjectKey{Name: node}, kubernetesNode)
		if err != nil {
			return err
		}

		// Shutdown the node
		fmt.Println("Shutting down the node")
		powerCheckCmd := exec.Command("neco", "power", "status", node)
		var out bytes.Buffer
		powerCheckCmd.Stdout = &out
		err = powerCheckCmd.Run()
		if err != nil {
			return err
		}
		if strings.TrimSpace(out.String()) == "On" {
			poweroffCmd := exec.Command("neco", "power", "stop", node)
			err = poweroffCmd.Run()
			if err != nil {
				return err
			}
			//wait for the node to be down
			fmt.Printf("Waiting for the node %s to be down\n", node)
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			for {
				select {
				case <-timeoutCtx.Done():
					return errors.New("power check timed out")
				default:
				}
				out.Reset()
				powerCheckCmd := exec.Command("neco", "power", "status", node)
				powerCheckCmd.Stdout = &out
				err = powerCheckCmd.Run()
				if err != nil {
					return err
				}
				if strings.TrimSpace(out.String()) == "Off" {
					break
				}
				time.Sleep(5 * time.Second)
			}
		}

		// Create NetworkFence for ceph clusters
		fmt.Println("Creating NetworkFence for ceph clusters")
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
				networkFence := csiaddonsv1alpha1.NetworkFence{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fenceName,
						Namespace: cephClusterName,
					},
					Spec: csiaddonsv1alpha1.NetworkFenceSpec{
						FenceState: csiaddonsv1alpha1.Fenced,
						Driver:     cephClusterName + ".rbd.csi.ceph.com",
						Cidrs:      []string{node + "/32"},
						Secret: csiaddonsv1alpha1.SecretSpec{
							Name:      "rook-csi-rbd-provisioner",
							Namespace: cephClusterName,
						},
						Parameters: map[string]string{
							"clusterID": cephClusterName,
						},
					},
				}
				err = kubeClient.Create(ctx, &networkFence)
				if err != nil {
					if client.IgnoreAlreadyExists(err) == nil {
						fmt.Printf("NetworkFence %s already exists\n", networkFence.Name)
					} else {
						return err
					}
				}
				// wait for fence of networkfence to be Succeeded
				timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				networkFence = csiaddonsv1alpha1.NetworkFence{}
				for {
					select {
					case <-timeoutCtx.Done():
						return errors.New("timeout waiting for networkfence to be fenced")
					default:
					}
					err := kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, &networkFence)
					if err != nil {
						return err
					}
					if networkFence.Status.Result == csiaddonsv1alpha1.FencingOperationResultSucceeded {
						break
					}
					time.Sleep(5 * time.Second)
				}
				return nil
			})
		}
		err = g.Wait()
		if err != nil {
			return err
		}

		// Add taint to the node
		fmt.Println("Adding taint to the node")
		tainted := false
		for _, taint := range kubernetesNode.Spec.Taints {
			if taint.Key == "node.kubernetes.io/out-of-service" {
				tainted = true
				break
			}
		}
		if !tainted {
			kubernetesNode.Spec.Taints = append(kubernetesNode.Spec.Taints, corev1.Taint{
				Key:    "node.kubernetes.io/out-of-service",
				Value:  "nodeshutdown",
				Effect: "NoExecute",
			})
			err = kubeClient.Update(ctx, kubernetesNode)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	nonGracefulNodeShutdownCmd.AddCommand(nonGracefulNodeShutdownShutdownCmd)
}
