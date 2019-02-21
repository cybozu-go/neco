package serf

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cybozu-go/neco"
)

func TestGenerateConf(t *testing.T) {
	t.Parallel()
	osName := "Ubuntu"
	osVer := "18.04"
	serial := "abc"
	expectedRetryJoin := []string{
		neco.BootNode0IP(0).String(),
		neco.BootNode0IP(1).String(),
		neco.BootNode0IP(2).String(),
	}
	expected := serfConfig{
		Tags: tags{
			OsName:     osName,
			OsVersion:  osVer,
			Serial:     serial,
			BootServer: "true",
		},
		Interface:         "node0",
		EventHandlers:     []string{"member-join,member-failed,member-leave=" + neco.SerfHandler},
		ReconnectInterval: "30s",
		ReconnectTimeout:  "24h",
		TombstoneTimeout:  "24h",
		RetryJoin:         expectedRetryJoin,
		RetryMaxAttempts:  0,
		RetryInterval:     "30s",
		LogLevel:          "debug",
		Protocol:          protocolVersion,
	}

	buf := new(bytes.Buffer)
	err := GenerateConf(buf, []int{0, 1, 2}, osName, osVer, serial)
	if err != nil {
		t.Fatal(err)
	}
	var actual serfConfig
	err = json.Unmarshal(buf.Bytes(), &actual)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("unexpected config: %#v", actual)
	}
}
