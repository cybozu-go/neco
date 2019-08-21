package etcd

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, 0, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}

	data := make(map[string]interface{})
	err = yaml.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(data["name"], "boot-0") {
		t.Error("name should be 'boot-0', actual: ", data["name"])
	}
	if !cmp.Equal(data["initial-advertise-peer-urls"], "https://10.69.0.3:2380") {
		t.Error("initial-advertise-peer-urls should be 'https://10.69.0.3:2380', actual: ", data["initial-advertise-peer-urls"])
	}
	if !cmp.Equal(data["advertise-client-urls"], "https://10.69.0.3:2379") {
		t.Error("advertise-client-urls should be 'https://10.69.0.3:2379', actual: ", data["advertise-client-urls"])
	}
	if !cmp.Equal(data["initial-cluster"], "boot-0=https://10.69.0.3:2380,boot-1=https://10.69.0.195:2380,boot-2=https://10.69.1.131:2380") {
		t.Error("initial-cluster should be 'boot-0=https://10.69.0.3:2380,boot-1=https://10.69.0.195:2380,boot-2=https://10.69.1.131:2380', actual: ", data["initial-cluster"])
	}
	if !cmp.Equal(data["initial-cluster-state"], "new") {
		t.Error("initial-cluster-state should be 'new', actual: ", data["initial-cluster-state"])
	}
}
