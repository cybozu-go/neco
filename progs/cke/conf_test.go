package cke

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	yaml "gopkg.in/yaml.v2"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}

	var actual map[string]interface{}
	err = yaml.Unmarshal(buf.Bytes(), &actual)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{
		"endpoints": []interface{}{
			"https://" + neco.BootNode0IP(0).String() + ":2379",
			"https://" + neco.BootNode0IP(1).String() + ":2379",
			"https://" + neco.BootNode0IP(2).String() + ":2379",
		},
		"tls-cert-file": neco.CKECertFile,
		"tls-key-file":  neco.CKEKeyFile,
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("unexpected config: %#v", actual)
	}
}

func TestGenerateCKETemplate(t *testing.T) {
	t.Parallel()

	defaultTemplate := `
taint_control_plane: false
service_subnet: ""
pod_subnet: ""
dns_servers: []
dns_service: ""
etcd_backup:
  enabled: false
  pvc_name: ""
  schedule: ""
  rotate: 14
options:
  etcd:
    extra_args: []
    extra_binds: []
    extra_env: {}
    volume_name: etcd-cke
  rivers:
    extra_args: []
    extra_binds: []
    extra_env: {}
  etcd-rivers:
    extra_args: []
    extra_binds: []
    extra_env: {}
  kube-api:
    extra_args: []
    extra_binds: []
    extra_env: {}
    audit_log_enabled: false
    audit_log_policy: ""
  kube-controller-manager:
    extra_args: []
    extra_binds: []
    extra_env: {}
  kube-scheduler:
    extra_args: []
    extra_binds: []
    extra_env: {}
    extenders: []
  kube-proxy:
    extra_args: []
    extra_binds: []
    extra_env: {}
  kubelet:
    extra_args: []
    extra_binds: []
    extra_env: {}
    container_runtime: ""
    container_runtime_endpoint: ""
    container_log_max_size: ""
    container_log_max_files: 0
    domain: cluster.local
    allow_swap: false
    boot_taints: []
    cni_conf_file:
      name: ""
      content: ""
`

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	st := storage.NewStorage(etcd)
	ctx := context.Background()

	ckeTemplate := `
name: test
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: "cs"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: "cs"
    cke.cybozu.com/weight: "18"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: "ss"
    cke.cybozu.com/weight: "10"
`

	expectedTemplate := `name: test
nodes:
- address: ""
  hostname: ""
  user: cybozu
  control_plane: true
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "18"
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "10"
  taints: []` + defaultTemplate

	out, err := GenerateCKETemplate(ctx, st, []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}

	if expectedTemplate != string(out) {
		t.Errorf("unexpected file content: %v", diff.LineDiff(string(expectedTemplate), string(out)))
	}

	err = st.PutCKEWeight(ctx, map[string]float64{
		"foo": float64(100),
	})
	if err != nil {
		t.Error(err)
	}

	out, err = GenerateCKETemplate(ctx, st, []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}

	if expectedTemplate != string(out) {
		t.Errorf("unexpected cke template content: %v", diff.LineDiff(string(expectedTemplate), string(out)))
	}

	expectedTemplate = `name: test
nodes:
- address: ""
  hostname: ""
  user: cybozu
  control_plane: true
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "10.000000"
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "20.000000"
  taints: []` + defaultTemplate

	st.PutCKEWeight(ctx, map[string]float64{
		"cs": float64(10),
		"ss": float64(20),
	})

	out, err = GenerateCKETemplate(ctx, st, []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}

	if expectedTemplate != string(out) {
		t.Errorf("unexpected cke template content: %v", diff.LineDiff(string(expectedTemplate), string(out)))
	}

	ckeTemplate = `
name: test
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: "cs"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: "cs"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: "ss"
`

	expectedTemplate = `name: test
nodes:
- address: ""
  hostname: ""
  user: cybozu
  control_plane: true
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "33.333333"
  taints: []
- address: ""
  hostname: ""
  user: cybozu
  control_plane: false
  annotations: {}
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "11.111111"
  taints: []` + defaultTemplate

	st.PutCKEWeight(ctx, map[string]float64{
		"cs": float64(33.333333),
		"ss": float64(11.111111),
	})

	out, err = GenerateCKETemplate(ctx, st, []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}

	if expectedTemplate != string(out) {
		t.Errorf("unexpected cke template content: %v", diff.LineDiff(string(expectedTemplate), string(out)))
	}
}
