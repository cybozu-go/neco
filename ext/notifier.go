package ext

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

// Notifier notifies the result of update to the outside.
type Notifier interface {
	NotifyInfo(req neco.UpdateRequest, message string) error
	NotifySucceeded(req neco.UpdateRequest) error
	NotifyFailure(req neco.UpdateRequest, message string) error
}

type nopNotifier struct {
}

func (n nopNotifier) NotifyInfo(req neco.UpdateRequest, message string) error {
	return nil
}
func (n nopNotifier) NotifySucceeded(req neco.UpdateRequest) error {
	return nil
}
func (n nopNotifier) NotifyFailure(req neco.UpdateRequest, message string) error {
	return nil
}

// NewNotifier creates a new Notifier.
func NewNotifier(ctx context.Context, st storage.Storage) (Notifier, error) {
	slackURL, err := st.GetSlackNotification(ctx)
	if err == storage.ErrNotFound {
		return nopNotifier{}, nil
	}

	if err != nil {
		return nil, err
	}

	hc, err := ProxyHTTPClient(ctx, st)
	if err != nil {
		return nil, err
	}

	me, err := neco.MyCluster()
	if err != nil {
		return nil, err
	}

	return &SlackClient{URL: slackURL, HTTP: hc, Cluster: me}, nil
}
