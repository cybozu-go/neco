package cmd

import (
	"fmt"
	"os"

	"github.com/cybozu-go/neco/pkg/ingress-watcher/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

var registry *prometheus.Registry

var rootCmd = &cobra.Command{
	Use:   "ingress-watcher",
	Short: "Ingress monitoring tool for Neco",
	Long:  `Ingress monitoring tool for Neco.`,
}

// Execute executes the command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	registry = prometheus.NewRegistry()
	registry.MustRegister(
		metrics.WatchInterval,
		metrics.HTTPGetTotal,
		metrics.HTTPGetSuccessfulTotal,
		metrics.HTTPGetFailTotal,
	)
}
