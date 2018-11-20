package serf

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/cybozu-go/neco"
)

// GenerateConf generates serf.json from template.
func GenerateConf(w io.Writer, lrns []int) error {
	endpoints := make([]string, len(lrns))
	for i, lrn := range lrns {
		endpoints[i] = neco.BootNode0IP(lrn).String()
	}

	osVer, err := getOSVersion()
	if err != nil {
		return err
	}
	serial, err := exec.Command("cat", "/sys/class/dmi/id/product_serial").Output()
	if err != nil {
		return err
	}

	data := confTmpl{
		Tags: tags{
			OsVersion: osVer,
			Serial:    string(serial),
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

func getOSVersion() (string, error) {
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
