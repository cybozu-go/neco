package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cybozu-go/neco"
	necorebooter "github.com/cybozu-go/neco/pkg/neco-rebooter"
	"github.com/cybozu-go/neco/storage"
)

var (
	flagCKEConfig     = flag.String("cke-config", neco.CKEConfFile, "path of cke config file")
	flagConfigFile    = flag.String("config", neco.NecoRebooterConfFile, "path of config file")
	errSignalReceived = errors.New("signal received")
)

func main() {
	flag.Parse()

	configFile, err := os.Open(*flagConfigFile)
	if err != nil {
		slog.Error("failed to open config file", "err", err)
		os.Exit(1)
	}
	defer configFile.Close()

	c, err := necorebooter.LoadConfig(configFile)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	t, err := c.GetRebootTime()
	if err != nil {
		slog.Error("failed to get reboot time", "err", err)
		os.Exit(1)
	}

	ckeConfigFile, err := os.Open(*flagCKEConfig)
	if err != nil {
		slog.Error("failed to open cke config file", "err", err)
		os.Exit(1)
	}
	defer ckeConfigFile.Close()

	cs, err := necorebooter.NewCKEStorage(ckeConfigFile)
	if err != nil {
		slog.Error("failed to create cke storage", "err", err)
		os.Exit(1)
	}
	etcd, err := neco.EtcdClient()
	if err != nil {
		slog.Error("failed to create etcd client", "err", err)
		os.Exit(1)
	}
	ns := storage.NewStorage(etcd)

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("failed to get hostname", "err", err)
		os.Exit(1)
	}

	ctrl, err := necorebooter.NewController(c, t, cs, etcd, &ns, hostname)
	if err != nil {
		slog.Error("failed to create controller", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChannel
		fmt.Println("got signal")
		cancel(errSignalReceived)
	}()

	collector := necorebooter.NewCollector(ns, hostname)
	metricsHandler := necorebooter.GetMetricsHandler(collector)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", c.MetricsPort),
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)
	srv.Handler = mux
	go func() {
		slog.Info("starting metrics server...", "port", c.MetricsPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed to start metrics server", "err", err)
		}
	}()

	go func() {
		slog.Info("starting controller...")
		err = ctrl.Run(ctx)
		if err != nil {
			cancel(err)
		}
	}()

	<-ctx.Done()
	srv.Shutdown(ctx)
	slog.Info("shutting down metrics server...")
	if context.Cause(ctx) == errSignalReceived {
		slog.Info("exit by signal, waiting for 5 seconds...")
		<-time.After(5 * time.Second)
		os.Exit(0)
	} else {
		slog.Error("exit by error, waiting for 5 seconds...", "err", context.Cause(ctx))
		<-time.After(5 * time.Second)
		os.Exit(1)
	}
}
