package updater

import (
	"context"

	"github.com/cybozu-go/neco"
)

// Notifier notifies the result of update to the outside.
type Notifier interface {
	NotifySucceeded(ctx context.Context, req neco.UpdateRequest) error
	NotifyServerFailure(ctx context.Context, req neco.UpdateRequest, message string) error
	NotifyTimeout(ctx context.Context, req neco.UpdateRequest) error
}

type nopNotifier struct {
}

func (n nopNotifier) NotifySucceeded(ctx context.Context, req neco.UpdateRequest) error {
	return nil
}
func (n nopNotifier) NotifyServerFailure(ctx context.Context, req neco.UpdateRequest, message string) error {
	return nil
}
func (n nopNotifier) NotifyTimeout(ctx context.Context, req neco.UpdateRequest) error {
	return nil
}
