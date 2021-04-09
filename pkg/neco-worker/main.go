package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/worker"
	"github.com/cybozu-go/well"
)

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	ec, err := neco.EtcdClient()
	if err != nil {
		log.ErrorExit(err)
	}
	defer ec.Close()

	version, err := neco.GetDebianVersion(neco.NecoPackageName)
	if err != nil {
		log.ErrorExit(err)
	}
	log.Info("neco package version", map[string]interface{}{
		"version": version,
	})

	mylrn, err := neco.MyLRN()
	if err != nil {
		log.ErrorExit(err)
	}

	well.Go(func(ctx context.Context) error {
		st := storage.NewStorage(ec)
		proxy, err := st.GetProxyConfig(ctx)
		if err != nil && err != storage.ErrNotFound {
			return err
		}
		if err := configureSystemProxy(ctx, proxy); err != nil {
			return err
		}

		if err := configureSystemDNS(ctx); err != nil {
			return err
		}

		op, err := worker.NewOperator(ctx, ec, mylrn)
		if err != nil {
			return err
		}
		w := worker.NewWorker(ec, op, version, mylrn)
		return w.Run(ctx)
	})
	well.Go(storage.NewStorage(ec).WaitConfigChange)
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

func configureSystemProxy(ctx context.Context, proxy string) error {
	if proxy == "" {
		return nil
	}

	out, err := exec.Command("docker", "info", "-f", "{{.HTTPProxy}}").Output()
	if err != nil {
		return fmt.Errorf("failed to invoke docker info: %w", err)
	}
	if strings.TrimSpace(string(out)) == proxy {
		return nil
	}

	dir := "/etc/systemd/system/docker.service.d"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, "override.conf"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create override conf for docker: %w", err)
	}
	defer f.Close()

	contents := fmt.Sprintf(`[Service]
Environment="HTTP_PROXY=%s"
Environment="HTTPS_PROXY=%s"
`, proxy, proxy)

	if _, err := io.WriteString(f, contents); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	return neco.RestartService(ctx, "docker")
}

func configureSystemDNS(ctx context.Context) error {
	if hasResolved, _ := neco.IsActiveService(ctx, "systemd-resolved"); hasResolved {
		return errors.New("systemd-resolved.service is running")
	}

	resolvconf := "/etc/resolv.conf"
	contents := `nameserver 127.0.0.1
search cluster.local
options ndots:3
`
	if err := os.WriteFile(resolvconf+".tmp", []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to create /etc/resolv.conf.tmp: %w", err)
	}

	if err := os.Rename(resolvconf+".tmp", resolvconf); err != nil {
		return fmt.Errorf("failed to replace /etc/resolv.conf: %w", err)
	}
	return exec.Command("sync").Run()
}
