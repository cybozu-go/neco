package etcdbackup

import (
	"bytes"
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupPodUpdateOp struct {
	apiserver *cke.Node
	pvcname   string
	finished  bool
}

// PodUpdateOp returns an Operator to Update etcdbackup pod.
func PodUpdateOp(apiserver *cke.Node, pvcname string) cke.Operator {
	return &etcdBackupPodUpdateOp{
		apiserver: apiserver,
		pvcname:   pvcname,
	}
}

func (o *etcdBackupPodUpdateOp) Name() string {
	return "etcdbackup-pod-update"
}

func (o *etcdBackupPodUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateEtcdBackupPodCommand{o.apiserver, o.pvcname}
}

func (o *etcdBackupPodUpdateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type updateEtcdBackupPodCommand struct {
	apiserver *cke.Node
	pvcname   string
}

func (c updateEtcdBackupPodCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	claims := cs.CoreV1().PersistentVolumeClaims("kube-system")
	_, err = claims.Get(c.pvcname, metav1.GetOptions{})
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = podTemplate.Execute(buf, struct {
		PVCName string
	}{
		PVCName: c.pvcname,
	})
	if err != nil {
		return err
	}

	pod := new(corev1.Pod)
	err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(pod)
	if err != nil {
		return err
	}

	pods := cs.CoreV1().Pods("kube-system")
	_, err = pods.Update(pod)
	return err
}

func (c updateEtcdBackupPodCommand) Command() cke.Command {
	return cke.Command{
		Name:   "update-etcdbackup-pod",
		Target: "etcdbackup",
	}
}
