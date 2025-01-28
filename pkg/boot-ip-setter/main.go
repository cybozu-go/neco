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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

const (
	defaultInterface     = "boot"
	defaultInterval      = 1 * time.Minute
	defaultListenAddress = "0.0.0.0:4192"
)

var (
	flagDebugLog      = flag.Bool("debug", false, "Show debug log or not.")
	flagInterface     = flag.String("interface", defaultInterface, "The target network interface that this program operates.")
	flagInterval      = flag.Duration("interval", defaultInterval, "The interval for periodic operation.")
	flagListenAddress = flag.String("listen-addr", defaultListenAddress, "The listen address.")
)

func main() {
	flag.Parse()

	logLevel := new(slog.LevelVar)
	if *flagDebugLog {
		logLevel.Set(slog.LevelDebug)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	logger.Info("boot-ip-setter has started", "interface", *flagInterface, "interval", *flagInterval, "listen address", *flagListenAddress)

	netif := NewInterface(*flagInterface)
	err := subMain(ctx, logger, netif, *flagInterval, *flagListenAddress)
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("boot-ip-setter has finished abnormally", "error", err)

		// delete all addresses if this program ends abnormally
		err := netif.DeleteAllAddr()
		if err != nil {
			logger.Error("failed to delete address on exit", "error", err)
		}
		err = netif.Down()
		if err != nil {
			logger.Error("failed to down interface on exit", "error", err)
		}

		os.Exit(1)
	}

	logger.Info("boot-ip-setter has finished")
}

func subMain(ctx context.Context, logger *slog.Logger, netif NetworkInterface, interval time.Duration, listenAddr string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}
	logger.Debug("succeeded to get hostname", "hostname", hostname)

	rack, err := neco.MyLRN()
	if err != nil {
		return fmt.Errorf("failed to get logical rack number: %w", err)
	}
	logger.Debug("succeeded to get logical rack number", "rack", rack)

	etcdClient, err := neco.EtcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer etcdClient.Close()
	logger.Debug("succeeded to create etcd client")

	errorCounter := &atomic.Int32{}

	hostnameHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, hostname)
	})

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		newCollector(logger.With("component", "metrics collector"), hostname, netif, errorCounter),
	)
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true})

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		logger.Info("starting ip setter")
		return runIPSetter(ctx, logger.With("component", "ip setter"), etcdClient, netif, errorCounter, interval, rack)
	})

	eg.Go(func() error {
		logger.Info("starting http server")
		return runHTTPServer(ctx, listenAddr, hostnameHandler, metricsHandler)
	})

	return eg.Wait()
}

func runHTTPServer(ctx context.Context, listenAddr string, hostnameHandler, metricsHandler http.Handler) error {
	mux := http.NewServeMux()
	mux.Handle("/hostname", hostnameHandler)
	mux.Handle("/metrics", metricsHandler)
	server := &http.Server{Addr: listenAddr, Handler: mux}

	errCh := make(chan error)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		// ListenAndServe always returns a non-nil error. So no need for a nil check.
		return fmt.Errorf("failed to listen: %w", err)
	case <-ctx.Done():
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx2); err != nil {
			return fmt.Errorf("failed to shutdown: %w", err)
		}
		return ctx.Err()
	}
}
