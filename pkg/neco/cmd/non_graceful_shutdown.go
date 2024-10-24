package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nonGracefulNodeShutdownConfig = "/tmp/non-graceful-node-shutdown-config"
)

var nonGracefulNodeShutdownCmd = &cobra.Command{
	Use:   "non-graceful-node-shutdown",
	Short: "non-Graceful Node Shutdown related commands",
	Long:  `non-Graceful Node Shutdown related commands.`,
}

type CephCluster struct {
	Name      string
	NameSpace string
}

func issueAndLoadKubeconfigForNonGracefulNodeShutdown() (client.Client, error) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	csiaddonsv1alpha1.AddToScheme(scheme)

	stdout, err := os.Create(nonGracefulNodeShutdownConfig)
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	issueCmd := exec.Command(neco.CKECLIBin, "kubernetes", "issue")
	issueCmd.Stdout = stdout
	err = issueCmd.Run()
	if err != nil {
		return nil, err
	}
	stdout.Sync()

	config, err := clientcmd.BuildConfigFromFlags("", nonGracefulNodeShutdownConfig)
	if err != nil {
		return nil, err
	}

	kubeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func listRBDCephClusters(ctx context.Context, kubeClient client.Client) ([]CephCluster, error) {
	cephClusters := []CephCluster{}
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
		cephClusters = append(cephClusters, CephCluster{Name: cephCluster.Items[0].GetName(), NameSpace: cephCluster.Items[0].GetNamespace()})
	}
	return cephClusters, nil
}

func generateFenceName(clusterName, node string) string {
	return clusterName + "-" + strings.Replace(node, ".", "-", -1)
}

func init() {
	rootCmd.AddCommand(nonGracefulNodeShutdownCmd)
}
