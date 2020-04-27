package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupConfigMapCreateOp struct {
	apiserver *cke.Node
	rotate    int
	finished  bool
}

// ConfigMapCreateOp returns an Operator to create etcdbackup config.
func ConfigMapCreateOp(apiserver *cke.Node, rotate int) cke.Operator {
	return &etcdBackupConfigMapCreateOp{
		apiserver: apiserver,
		rotate:    rotate,
	}
}

func (o *etcdBackupConfigMapCreateOp) Name() string {
	return "etcdbackup-configmap-create"
}

func (o *etcdBackupConfigMapCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupConfigMapCommand{o.apiserver, o.rotate}
}

func (o *etcdBackupConfigMapCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdBackupConfigMapCommand struct {
	apiserver *cke.Node
	rotate    int
}

func (c createEtcdBackupConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(op.EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = configs.Create(RenderConfigMap(c.rotate))
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

// RenderConfigMap returns ConfigMap for etcdbackup
func RenderConfigMap(rotate int) *corev1.ConfigMap {
	config := new(corev1.ConfigMap)
	buf := new(bytes.Buffer)
	err := configMapTemplate.Execute(buf, struct {
		Rotate int
	}{
		rotate,
	})
	if err != nil {
		panic(err)
	}
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(config)
	if err != nil {
		panic(err)
	}
	return config
}

func (c createEtcdBackupConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcdbackup-configmap",
		Target: "etcdbackup",
	}
}
