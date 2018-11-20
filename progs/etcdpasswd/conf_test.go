package etcdpasswd

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/cybozu-go/neco"
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
		"tls-cert-file": neco.EtcdpasswdCertFile,
		"tls-key-file":  neco.EtcdpasswdKeyFile,
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("unexpected config: %#v", actual)
	}

}
