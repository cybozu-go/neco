package op

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubeWaitOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeWaitOp returns an Operator to wait for Kubernetes resources gets initialized
func KubeWaitOp(apiserver *cke.Node) cke.Operator {
	return &kubeWaitOp{apiserver: apiserver}
}

func (o *kubeWaitOp) Name() string {
	return "wait-kubernetes"
}

func (o *kubeWaitOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return waitKubeCommand{o.apiserver}
}

func (o *kubeWaitOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type waitKubeCommand struct {
	apiserver *cke.Node
}

func (c waitKubeCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	begin := time.Now()
	for i := 0; i < 100; i++ {
		_, err = cs.CoreV1().ServiceAccounts("kube-system").Get("default", metav1.GetOptions{})
		switch {
		case err == nil:
			elapsed := time.Now().Sub(begin)
			log.Info("k8s gets initialized", map[string]interface{}{
				"elapsed": elapsed.Seconds(),
			})
			return nil

		case errors.IsNotFound(err):
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
			}

		default:
			return err
		}
	}

	// Timed-out here is not an error because waitKubeCommand will be invoked
	// again by the controller.
	return nil
}

func (c waitKubeCommand) Command() cke.Command {
	return cke.Command{
		Name:   "waitKubeCommand",
		Target: "kube-system sa/default",
	}
}
