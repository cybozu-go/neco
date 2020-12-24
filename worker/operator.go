package worker

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/progs/systemdresolved"
	"github.com/cybozu-go/neco/storage"
)

// Operator installs or updates programs
type Operator interface {
	// UpdateNeco updates neco package.
	UpdateNeco(ctx context.Context, req *neco.UpdateRequest) error

	// FinalStep is the step number of the final operation.
	FinalStep() int

	// RunStep executes operations for given step.
	RunStep(ctx context.Context, req *neco.UpdateRequest, step int) error

	// RestoreServices starts installed services at startup.
	StartServices(ctx context.Context) error

	// RestartEtcd restarts etcd in case it is necessary.
	RestartEtcd(index int, req *neco.UpdateRequest) error

	// ReplaceSystemdResolvedFiles restarts systemd-resolved for ingress-watcher
	ReplaceSystemdResolvedFiles(ctx context.Context) error
}

type operator struct {
	mylrn            int
	ec               *clientv3.Client
	storage          storage.Storage
	ghClient         *http.Client
	proxyClient      *http.Client
	localClient      *http.Client
	containerRuntime neco.ContainerRuntime

	etcdRestart bool
}

// NewOperator creates an Operator
func NewOperator(ctx context.Context, ec *clientv3.Client, mylrn int) (Operator, error) {
	st := storage.NewStorage(ec)
	localClient := ext.LocalHTTPClient()
	proxyClient, err := ext.ProxyHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}
	ghClient, err := ext.GitHubHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}
	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		return nil, err
	}
	rt, err := neco.GetContainerRuntime(proxy)
	if err != nil {
		return nil, err
	}

	return &operator{
		mylrn:            mylrn,
		ec:               ec,
		storage:          st,
		ghClient:         ghClient,
		proxyClient:      proxyClient,
		localClient:      localClient,
		containerRuntime: rt,
	}, nil
}

func (o *operator) UpdateNeco(ctx context.Context, req *neco.UpdateRequest) error {
	deb := &neco.DebianPackage{
		Name:       neco.NecoPackageName,
		Repository: neco.GitHubRepoName,
		Owner:      neco.GitHubRepoOwner,
		Release:    "release-" + req.Version,
	}

	log.Info("update neco", map[string]interface{}{
		"version": req.Version,
	})
	env, err := o.storage.GetEnvConfig(ctx)
	if err != nil {
		return err
	}
	if env == neco.TestEnv {
		return installLocalPackage(ctx, deb)
	}
	return InstallDebianPackage(ctx, o.proxyClient, o.ghClient, deb, true)
}

func (o *operator) FinalStep() int {
	return 19
}

func (o *operator) RunStep(ctx context.Context, req *neco.UpdateRequest, step int) error {

	switch step {
	case 1:
		return o.FetchImages(ctx, req)
	case 2:
		return o.UpdateEtcd(ctx, req)
	case 3:
		return o.StopVault(ctx, req)
	case 4:
		return o.UpdateVault(ctx, req)
	case 5:
		return o.UpdateSetupHW(ctx, req)
	case 6:
		return o.UpdateSerf(ctx, req)
	case 7:
		return o.UpdateSetupSerfTags(ctx, req)
	case 8:
		return o.UpdateEtcdpasswd(ctx, req)
	case 9:
		return o.UpdateSabakan(ctx, req)
	case 10:
		return o.UpdateSabakanStateSetter(ctx, req)
	case 11:
		return o.StopCKE(ctx, req)
	case 12:
		return o.UpdateCKE(ctx, req)
	case 13:
		return o.UpdateCKEContents(ctx, req)
	case 14:
		return o.UpdateSabakanContents(ctx, req)
	case 15:
		return o.UpdateDHCPJSON(ctx, req)
	case 16:
		return o.UpdateCKETemplate(ctx, req)
	case 17:
		return o.UpdateUserResources(ctx, req)
	case 18:
		return o.UpdateIngressWatcher(ctx, req)
	case 19:
		// THIS MUST BE THE FINAL STEP!!!!!
		// to synchronize before restarting etcd.
		return nil
		// DO NOT ADD ANY STEP AFTER THIS LINE!!!
	}

	return fmt.Errorf("invalid step: %d", step)
}

func (o *operator) restoreService(ctx context.Context, svc string) error {
	_, err := os.Stat(neco.ServiceFile(svc))
	if err != nil {
		// lint:ignore nilerr  Do nothing if service file does not exist.
		return nil
	}

	return neco.StartService(ctx, svc)
}

func (o *operator) stopService(ctx context.Context, svc string) error {
	_, err := os.Stat(neco.ServiceFile(svc))
	if err != nil {
		// lint:ignore nilerr  Do nothing if service file does not exist.
		return nil
	}

	return neco.StopService(ctx, svc)
}

func (o *operator) StartServices(ctx context.Context) error {
	err := o.restoreService(ctx, neco.EtcdService)
	if err != nil {
		return err
	}
	err = o.restoreService(ctx, neco.VaultService)
	if err != nil {
		return err
	}

	return neco.RestartService(ctx, neco.SystemdResolvedService)
}

func (o *operator) ReplaceSystemdResolvedFiles(ctx context.Context) error {
	dnsAddress, err := o.storage.GetDNSConfig(ctx)
	if err == storage.ErrNotFound {
		log.Info("systemd-resolved: dns config not found", nil)
		return nil
	}
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(neco.SystemdResolvedConfFile), 0755)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = systemdresolved.GenerateConfBase(buf, dnsAddress)
	if err != nil {
		return err
	}

	_, err = replaceFile(neco.SystemdResolvedConfFile, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
