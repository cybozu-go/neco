package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/neco"
	necorebooter "github.com/cybozu-go/neco/pkg/neco-rebooter"
	"github.com/cybozu-go/neco/storage"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	flagCKEConfig  string
	flagConfigFile string
	flagKubeconfig string
	necoStorage    storage.Storage
	ckeStorage     cke.Storage
	config         *necorebooter.Config
	KubeClient     kubernetes.Clientset
)

var rebooterCmd = &cobra.Command{
	Use:   "rebooter",
	Short: "neco-rebooter subcommand",
	Long:  `neco-rebooter subcommand.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var err error
		configFile, err := os.Open(flagConfigFile)
		if err != nil {
			return err
		}
		defer configFile.Close()
		config, err = necorebooter.LoadConfig(configFile)
		if err != nil {
			return err
		}
		ckeConfigFile, err := os.Open(flagCKEConfig)
		if err != nil {
			return err
		}
		defer ckeConfigFile.Close()
		cs, err := necorebooter.NewCKEStorage(ckeConfigFile)
		if err != nil {
			return err
		}
		ckeStorage = *cs

		etcd, err := neco.EtcdClient()
		if err != nil {
			return err
		}
		necoStorage = storage.NewStorage(etcd)

		retryCount := 0
	RETRY:
		config, err := clientcmd.BuildConfigFromFlags("", flagKubeconfig)
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

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		KubeClient = *clientset
		return nil
	},
}

func renewKubeConfig() error {
	out, err := os.Create(flagKubeconfig)
	if err != nil {
		return err
	}
	defer out.Close()
	com := exec.Command(neco.CKECLIBin, "kubernetes", "issue")
	com.Stdout = out
	slog.Info("renewing kubeconfig...", slog.String("path", flagKubeconfig))
	err = com.Run()
	if err != nil {
		return err
	}
	return nil
}

func matchRebootTimes(node v1.Node, rebootTimes []necorebooter.RebootTimes) (*necorebooter.RebootTimes, error) {
REBOOT_TIMES_LOOP:
	for _, rt := range rebootTimes {
		for key, value := range rt.LabelSelector.MatchLabels {
			if node.ObjectMeta.Labels[key] != value {
				continue REBOOT_TIMES_LOOP
			}
		}
		return &rt, nil
	}
	return nil, fmt.Errorf("node: %s does not match any reboot time", node.Name)
}

func init() {
	rebooterCmd.PersistentFlags().StringVar(&flagCKEConfig, "cke-config", neco.CKEConfFile, "cke config file")
	rebooterCmd.PersistentFlags().StringVar(&flagConfigFile, "config", neco.NecoRebooterConfFile, "neco-rebooter config file")
	var DefaultKubeconfig string
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		DefaultKubeconfig = env
	} else if home := homedir.HomeDir(); home != "" {
		DefaultKubeconfig = filepath.Join(home, ".kube", "config")
	}
	rebooterCmd.PersistentFlags().StringVar(&flagKubeconfig, "kubeconfig", DefaultKubeconfig, "kubeconfig file")
	rootCmd.AddCommand(rebooterCmd)
}
