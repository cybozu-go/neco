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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cephClusters = []string{"ceph-canary-block", "ceph-dotcom-block-0", "ceph-poc", "ceph-ssd"}
)

var nonGracefulNodeShutdownCmd = &cobra.Command{
	Use:   "nonGracefulNodeShutdown IP_ADDRESS",
	Short: "nonGracefulNodeShutdown related commands",
	Long:  `nonGracefulNodeShutdown related commands.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node := args[0]

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		scheme := runtime.NewScheme()
		clientgoscheme.AddToScheme(scheme)
		csiaddonsv1alpha1.AddToScheme(scheme)

		//issue kubeconfig
		issueCmd := exec.Command("sh", "-c", "ckecli kubernetes issue > /home/cybozu/.kube/shutdown-config")
		err := issueCmd.Run()
		if err != nil {
			fmt.Println("Failed to issue kubeconfig")
			os.Exit(1)
		}

		// Load kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", "/home/cybozu/.kube/shutdown-config")
		if err != nil {
			return err
		}
		kubeClient, err := client.New(config, client.Options{Scheme: scheme})
		if err != nil {
			return err
		}

		// Get the node
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
		L:
			for {
				select {
				case <-timeoutCtx.Done():
					return errors.New("power check timeout")
				default:
					out.Reset()
					powerCheckCmd := exec.Command("neco", "power", "status", node)
					powerCheckCmd.Stdout = &out
					err = powerCheckCmd.Run()
					if err != nil {
						return err
					}
					if strings.TrimSpace(out.String()) == "Off" {
						break L
					}
					time.Sleep(5 * time.Second)
				}
			}
		}

		// Create NetworkFence for ceph clusters
		fmt.Println("Creating NetworkFence for ceph clusters")
		for _, cephCluster := range cephClusters {
			//check cephCluster exists
			nameSpace := &corev1.Namespace{}
			err := kubeClient.Get(ctx, client.ObjectKey{Name: cephCluster}, nameSpace)
			if err != nil {
				if client.IgnoreNotFound(err) == nil {
					fmt.Printf("Namespace %s does not found\n", nameSpace.Name)
					continue
				} else {
					return err
				}
			}
			fenceName := cephCluster + "-" + strings.Replace(node, ".", "-", -1)
			networkFence := csiaddonsv1alpha1.NetworkFence{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fenceName,
					Namespace: cephCluster,
				},
				Spec: csiaddonsv1alpha1.NetworkFenceSpec{
					FenceState: csiaddonsv1alpha1.Fenced,
					Driver:     cephCluster + ".rbd.csi.ceph.com",
					Cidrs:      []string{node + "/32"},
					Secret: csiaddonsv1alpha1.SecretSpec{
						Name:      "rook-csi-rbd-provisioner",
						Namespace: cephCluster,
					},
					Parameters: map[string]string{
						"clusterID": cephCluster,
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
		L2:
			for {
				select {
				case <-timeoutCtx.Done():
					return errors.New("timeout waiting for networkfence to be fenced")
				default:
					err := kubeClient.Get(ctx, client.ObjectKey{Name: fenceName}, &networkFence)
					if err != nil {
						return err
					}
					if networkFence.Status.Result == csiaddonsv1alpha1.FencingOperationResultSucceeded {
						break L2
					}
					time.Sleep(5 * time.Second)
				}
			}
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
	rootCmd.AddCommand(nonGracefulNodeShutdownCmd)
}
