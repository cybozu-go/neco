package worker

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
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
}

type operator struct {
	mylrn       int
	ec          *clientv3.Client
	storage     storage.Storage
	proxyClient *http.Client
	localClient *http.Client

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

	return &operator{
		mylrn:       mylrn,
		ec:          ec,
		storage:     st,
		proxyClient: proxyClient,
		localClient: localClient,
	}, nil
}

func (o *operator) UpdateNeco(ctx context.Context, req *neco.UpdateRequest) error {
	deb := &neco.DebianPackage{
		Name:       neco.NecoPackageName,
		Repository: neco.GitHubRepoName,
		Owner:      neco.GitHubRepoOwner,
		Release:    req.Version,
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
	return InstallDebianPackage(ctx, o.proxyClient, deb, true)
}

func (o *operator) FinalStep() int {
	return 12
}

func (o *operator) RunStep(ctx context.Context, req *neco.UpdateRequest, step int) error {

	switch step {
	case 1:
		return o.UpdateEtcd(ctx, req)
	case 2:
		return o.StopVault(ctx, req)
	case 3:
		return o.UpdateVault(ctx, req)
	case 4:
		return o.UpdateOMSA(ctx, req)
	case 5:
		return o.UpdateSerf(ctx, req)
	case 6:
		return o.UpdateEtcdpasswd(ctx, req)
	case 7:
		return o.UpdateSabakan(ctx, req)
	case 8:
		return o.UpdateSabakanContents(ctx, req)
	case 9:
		return o.StopCKE(ctx, req)
	case 10:
		return o.UpdateCKE(ctx, req)
	case 11:
		return o.UpdateCKEContents(ctx, req)
	case 12:
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
		return nil
	}

	return neco.StartService(ctx, svc)
}

func (o *operator) stopService(ctx context.Context, svc string) error {
	_, err := os.Stat(neco.ServiceFile(svc))
	if err != nil {
		return nil
	}

	return neco.StopService(ctx, svc)
}

func (o *operator) StartServices(ctx context.Context) error {
	err := o.restoreService(ctx, neco.EtcdService)
	if err != nil {
		return err
	}

	return o.restoreService(ctx, neco.VaultService)
}
