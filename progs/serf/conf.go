package serf

import (
	"encoding/json"
	"io"

	"github.com/cybozu-go/neco"
)

// NOTE: Administrator must consider protocol version in the Serf cluster on
// upgrading.  neco-worker currently does not support upgrading Serf and use
// fixed version number explicitly in the configuration for safety.
//
// See also https://www.serf.io/docs/upgrading.html
const protocolVersion = 5

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
		EventHandlers:     []string{"member-join,member-failed,member-leave=" + neco.SerfHandler},
		ReconnectInterval: "30s",
		ReconnectTimeout:  "24h",
		TombstoneTimeout:  "24h",
		RetryJoin:         endpoints,
		RetryMaxAttempts:  0,
		RetryInterval:     "30s",
		LogLevel:          "debug",
		Protocol:          protocolVersion,
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(data)
}
