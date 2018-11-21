package serf

import (
	"encoding/json"
	"io"

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
		EventHandlers:     []string{"member-join,member-failed,member-leave=/etc/serf/handler"},
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
