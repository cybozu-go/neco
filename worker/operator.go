package worker

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	"github.com/google/go-github/v50/github"
	clientv3 "go.etcd.io/etcd/client/v3"
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

	// RestartEtcd restarts etcd.
	RestartEtcd(index int, req *neco.UpdateRequest) error
}

type operator struct {
	mylrn            int
	ec               *clientv3.Client
	storage          storage.Storage
	ghClient         *github.Client
	proxyClient      *http.Client
	localClient      *http.Client
	fetcher          neco.ImageFetcher
	containerRuntime neco.ContainerRuntime
}

// NewOperator creates an Operator
func NewOperator(ctx context.Context, ec *clientv3.Client, mylrn int) (Operator, error) {
	st := storage.NewStorage(ec)
	localClient := ext.LocalHTTPClient()
	proxyClient, err := ext.ProxyHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}
	ghHttpClient, err := ext.GitHubHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}
	ghClient := neco.NewGitHubClient(ghHttpClient)
	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		return nil, err
	}

	env, err := st.GetEnvConfig(ctx)
	if err != nil {
		return nil, err
	}
	fetcher := neco.NewImageFetcher(proxyClient.Transport, env)

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
		fetcher:          fetcher,
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
		return installLocalPackage(ctx, deb, map[string]string{"HOME": "/home/cybozu"})
	}
	if env == neco.DevEnv {
		deb.Release = "test-" + req.Version
	}
	return InstallDebianPackage(ctx, o.proxyClient, o.ghClient, deb, true, map[string]string{"HOME": "/home/cybozu"})
}

func (o *operator) FinalStep() int {
	return 18
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
		return o.UpdatePromtail(ctx, req)
	case 17:
		return o.UpdateUserResources(ctx, req)
	case 18:
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
