package etcdutil

import (
	"net"
	"net/url"
	"time"

	"github.com/cybozu-go/log"
)

// reorderEndpoints work around etcd/issues/9949.
func reorderEndpoints(endpoints []string, timeout time.Duration) {
	dialer := net.Dialer{
		Timeout: timeout,
	}

	for i, endpoint := range endpoints {
		addr := endpoint
		u, err := url.Parse(endpoint)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			if u.Port() == "" {
				// http and https is available even with CGO_ENABLED=0.
				// https://golang.org/src/net/lookup.go#L44
				addr = net.JoinHostPort(u.Host, u.Scheme)
			} else {
				addr = u.Host
			}
		}
		c, err := dialer.Dial("tcp", addr)
		if err != nil {
			continue
		}
		c.Close()

		endpoints[i] = endpoints[0]
		endpoints[0] = endpoint
		return
	}

	log.Warn("no etcd endpoint can be connected", map[string]interface{}{
		"endpoints": endpoints,
	})
}
