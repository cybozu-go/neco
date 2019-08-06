package etcdbackup

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type etcdBackupSecretCreateOp struct {
	apiserver *cke.Node
	finished  bool
}

// SecretCreateOp returns an Operator to create etcdbackup certificates.
func SecretCreateOp(apiserver *cke.Node) cke.Operator {
	return &etcdBackupSecretCreateOp{
		apiserver: apiserver,
	}
}

func (o *etcdBackupSecretCreateOp) Name() string {
	return "etcdbackup-secret-create"
}

func (o *etcdBackupSecretCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createEtcdBackupSecretCommand{o.apiserver}
}

func (o *etcdBackupSecretCreateOp) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type createEtcdBackupSecretCommand struct {
	apiserver *cke.Node
}

func (c createEtcdBackupSecretCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	secrets := cs.CoreV1().Secrets("kube-system")
	_, err = secrets.Get(op.EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		crt, key, err := cke.EtcdCA{}.IssueForBackup(ctx, inf)
		if err != nil {
			return err
		}
		ca, err := inf.Storage().GetCACertificate(ctx, "server")
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		err = secretTemplate.Execute(buf, struct {
			Cert string
			Key  string
			CA   string
		}{
			Cert: base64.StdEncoding.EncodeToString([]byte(crt)),
			Key:  base64.StdEncoding.EncodeToString([]byte(key)),
			CA:   base64.StdEncoding.EncodeToString([]byte(ca)),
		})
		if err != nil {
			return err
		}

		secret := new(corev1.Secret)
		err = k8sYaml.NewYAMLToJSONDecoder(buf).Decode(secret)
		if err != nil {
			return err
		}
		_, err = secrets.Create(secret)
		if err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

func (c createEtcdBackupSecretCommand) Command() cke.Command {
	return cke.Command{
		Name:   "create-etcdbackup-secret",
		Target: "etcdbackup",
	}
}
