package op

import (
	"context"

	"github.com/cybozu-go/cke"
)

type resourceApplyOp struct {
	apiserver *cke.Node
	resource  cke.ResourceDefinition
	finished  bool
}

// ResourceApplyOp creates or updates a Kubernetes object.
func ResourceApplyOp(apiServer *cke.Node, resource cke.ResourceDefinition) cke.Operator {
	return &resourceApplyOp{
		apiserver: apiServer,
		resource:  resource,
	}
}

func (o *resourceApplyOp) Name() string {
	return "resource-apply"
}

func (o *resourceApplyOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return o
}

func (o *resourceApplyOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

func (o *resourceApplyOp) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, o.apiserver)
	if err != nil {
		return err
	}
	return cke.ApplyResource(cs, o.resource.Definition, o.resource.Revision)
}

func (o *resourceApplyOp) Command() cke.Command {
	return cke.Command{
		Name:   "apply-resource",
		Target: o.resource.String(),
	}
}
