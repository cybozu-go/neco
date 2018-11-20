package serf

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/cybozu-go/neco"
)

// GenerateConf generates serf.json from template.
func GenerateConf(w io.Writer, lrns []int, osVer string, serial string) error {
	endpoints := make([]string, len(lrns))
	for i, lrn := range lrns {
		endpoints[i] = neco.BootNode0IP(lrn).String()
	}

	data := serfConfig{
		Tags: tags{
			OsVersion: osVer,
			Serial:    serial,
		},
		Interface:         "node0",
		EventHandlers:     []string{"member-join", "member-failed", "member-leave=/etc/serf/handler"},
		ReconnectInterval: "30s",
		ReconnectTimeout:  "24h",
		TombstoneTimeout:  "24h",
		RetryJoin:         endpoints,
		RetryMaxAttempts:  0,
		RetryInterval:     "30s",
		LogLevel:          "debug",
	}
	return json.NewEncoder(w).Encode(data)
}

// GetOSVersion get OS version.
// Currently, this method is worked only on Linux.
func GetOSVersion() (string, error) {
	out, err := ioutil.ReadFile("/etc/lsb-release")
	if err != nil {
		return "", err
	}
	target := "DISTRIB_RELEASE="
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, target) {
			return line[len(target):], nil
		}
	}
	return "", errors.New("failed to get os version")
}
