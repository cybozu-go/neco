package cke

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
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
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: cs
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "18"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "10"`
	var expected map[string]interface{}
	err := yaml.Unmarshal([]byte(expectedTemplate), &expected)
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateCKETemplate(ctx, st, "test", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	var actual map[string]interface{}
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
	}

	err = st.PutCKEWeight(ctx, map[string]float64{
		"foo": float64(100),
	})
	if err != nil {
		t.Error(err)
	}
	out, err = GenerateCKETemplate(ctx, st, "test", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	actual = nil
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
	}

	expectedTemplate = `name: test
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: cs
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "10.000000"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "20.000000"`
	expected = nil
	err = yaml.Unmarshal([]byte(expectedTemplate), &expected)
	if err != nil {
		t.Fatal(err)
	}
	err = st.PutCKEWeight(ctx, map[string]float64{
		"cs": float64(10),
		"ss": float64(20),
	})
	if err != nil {
		t.Error(err)
	}
	out, err = GenerateCKETemplate(ctx, st, "test", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	actual = nil
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
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
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: cs
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: cs
    cke.cybozu.com/weight: "33.333333"
- user: cybozu
  control_plane: false
  labels:
    cke.cybozu.com/role: ss
    cke.cybozu.com/weight: "11.111111"`
	expected = nil
	err = yaml.Unmarshal([]byte(expectedTemplate), &expected)
	if err != nil {
		t.Fatal(err)
	}
	err = st.PutCKEWeight(ctx, map[string]float64{
		"cs": float64(33.333333),
		"ss": float64(11.111111),
	})
	if err != nil {
		t.Error(err)
	}
	out, err = GenerateCKETemplate(ctx, st, "test", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	actual = nil
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
	}
}

func TestGenerateCKETemplateReboot(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	st := storage.NewStorage(etcd)
	ctx := context.Background()

	ckeTemplate := `name: test
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: "cs"
reboot:
  command: ["test"]
`
	expectedTemplate := `name: test
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: "cs"
reboot:
  command: ["test"]
`
	var expected map[string]interface{}
	err := yaml.Unmarshal([]byte(expectedTemplate), &expected)
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateCKETemplate(ctx, st, "test", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	var actual map[string]interface{}
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
	}

	expectedTemplate = `name: stage0
nodes:
- user: cybozu
  control_plane: true
  labels:
    cke.cybozu.com/role: "cs"
reboot:
  command: ["test"]
  protected_namespaces:
    matchExpressions:
      - key: ignore-pdb
        operator: NotIn
        values: ["true"]
`
	err = yaml.Unmarshal([]byte(expectedTemplate), &expected)
	if err != nil {
		t.Fatal(err)
	}
	out, err = GenerateCKETemplate(ctx, st, "stage0", []byte(ckeTemplate))
	if err != nil {
		t.Error(err)
	}
	err = yaml.Unmarshal(out, &actual)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(expected, actual) {
		t.Error("unexpected file content:", cmp.Diff(expected, actual))
	}
}
