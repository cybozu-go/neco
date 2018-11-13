package ext

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cybozu-go/neco/storage"
)

// HTTPClient returns a *http.Client to access Internet.
func HTTPClient(ctx context.Context, st storage.Storage) (*http.Client, error) {
	proxyURL, err := st.GetProxyConfig(ctx)
	if err == storage.ErrNotFound {
		return http.DefaultClient, nil
	}
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(u),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   1 * time.Hour,
	}, nil
}
