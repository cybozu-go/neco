package ext

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cybozu-go/neco/storage"
	"golang.org/x/oauth2"
)

// ProxyHTTPClient returns a *http.Client to access Internet.
func ProxyHTTPClient(ctx context.Context, st storage.Storage) (*http.Client, error) {
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

// LocalHTTPClient returns a *http.Client to access intranet services.
func LocalHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil,
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
		Timeout:   10 * time.Minute,
	}
}

// GitHubHTTPClient returns a *http.Client to access Internet with GitHub personal access token.
// It returns *http.Client of ProxyHTTPClient() when token does not exist.
func GitHubHTTPClient(ctx context.Context, st storage.Storage) (*http.Client, error) {
	hc, err := ProxyHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}

	token, err := st.GetGitHubToken(ctx)
	if err == storage.ErrNotFound {
		return hc, nil
	}
	if err != nil {
		return nil, err
	}

	// Set personal access token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	// Add proxy http client to oauth2 generated http.Client
	ctx = context.WithValue(ctx, oauth2.HTTPClient, hc)
	// Create access token and proxy configuration included *http.Client
	return oauth2.NewClient(ctx, ts), nil
}
