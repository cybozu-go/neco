package cmd

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nonGracefulNodeShutdownConfig = "/tmp/non-graceful-node-shutdown-config"
)

var nonGracefulNodeShutdownCmd = &cobra.Command{
	Use:   "non-graceful-node-shutdown",
	Short: "Non-Graceful Node Shutdown related commands",
	Long:  `Non-Graceful Node Shutdown related commands.`,
}

func issueAndLoadKubeconfigForNonGracefulNodeShutdown() (client.Client, error) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	csiaddonsv1alpha1.AddToScheme(scheme)

	out, err := exec.Command(neco.CKECLIBin, "kubernetes", "issue").Output()
	if err != nil {
		return nil, err
	}
	kubeConfigGetter := func() (*clientcmdapi.Config, error) {
		return clientcmd.Load([]byte(out))
	}
	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", kubeConfigGetter)
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func listRBDCephClusters(ctx context.Context, kubeClient client.Client) ([]types.NamespacedName, error) {
	cephClusters := []types.NamespacedName{}
	scs := &storagev1.StorageClassList{}
	err := kubeClient.List(ctx, scs)
	if err != nil {
		return nil, err
	}
	cephClusterIDs := []string{}
	for _, sc := range scs.Items {
		if strings.HasSuffix(sc.Provisioner, "rbd.csi.ceph.com") {
			cephClusterIDs = append(cephClusterIDs, sc.Parameters["clusterID"])
		}
	}
	for _, cephClusterID := range cephClusterIDs {
		cephCluster := &unstructured.UnstructuredList{}
		cephCluster.SetAPIVersion("ceph.rook.io/v1")
		cephCluster.SetKind("CephCluster")
		err = kubeClient.List(ctx, cephCluster, &client.ListOptions{Namespace: cephClusterID})
		if err != nil {
			return nil, err
		}
		if len(cephCluster.Items) != 1 {
			return nil, errors.New("cephCluster is not found or multiple cephClusters are found")
		}
		cephClusters = append(cephClusters, types.NamespacedName{Name: cephCluster.Items[0].GetName(), Namespace: cephCluster.Items[0].GetNamespace()})
	}
	return cephClusters, nil
}

func generateFenceName(clusterName, node string) string {
	return clusterName + "-" + strings.Replace(node, ".", "-", -1)
}

func init() {
	rootCmd.AddCommand(nonGracefulNodeShutdownCmd)
}
