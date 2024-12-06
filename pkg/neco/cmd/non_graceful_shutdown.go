package cmd

import (
	"os"
	"os/exec"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cephClusters                  = []string{"ceph-canary-block", "ceph-dotcom-block-0", "ceph-poc", "ceph-ssd"}
	nonGracefulNodeShutdownConfig = "/tmp/non-graceful-node-shutdown-config"
)

var nonGracefulNodeShutdownCmd = &cobra.Command{
	Use:   "nonGracefulNodeShutdown",
	Short: "nonGracefulNodeShutdown related commands",
	Long:  `nonGracefulNodeShutdown related commands.`,
}

func IssueAndLoadKubeconfigForNonGracefulNodeShutdown() (client.Client, error) {
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

func init() {
	rootCmd.AddCommand(nonGracefulNodeShutdownCmd)
}
