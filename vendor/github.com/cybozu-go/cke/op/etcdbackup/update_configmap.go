package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupConfigMapUpdateOp struct {
	apiserver *cke.Node
	rotate    int
	finished  bool
}

// ConfigMapUpdateOp returns an Operator to Update etcdbackup ConfigMap.
func ConfigMapUpdateOp(apiserver *cke.Node, rotate int) cke.Operator {
	return &etcdBackupConfigMapUpdateOp{
		apiserver: apiserver,
		rotate:    rotate,
	}
}

func (o *etcdBackupConfigMapUpdateOp) Name() string {
	return "etcdbackup-configmap-update"
}

func (o *etcdBackupConfigMapUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateEtcdBackupConfigMapCommand{o.apiserver, o.rotate}
}

func (o *etcdBackupConfigMapUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type updateEtcdBackupConfigMapCommand struct {
	apiserver *cke.Node
	rotate    int
}

func (c updateEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = configMapTemplate.Execute(buf, struct {
		Rotate int
	}{
		Rotate: c.rotate,
	})
	if err != nil {
		return err
	}

	ConfigMap := new(corev1.ConfigMap)
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(ConfigMap)
	if err != nil {
		return err
	}

	maps := cs.CoreV1().ConfigMaps("kube-system")
	_, err = maps.Update(ConfigMap)
	return err
}

func (c updateEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "update-etcdbackup-configmap",
		Target: "etcdbackup",
	}
}
