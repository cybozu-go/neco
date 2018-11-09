package updater

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

type workerWatcher struct {
	current  *neco.UpdateRequest
	statuses map[int]*neco.UpdateStatus
	aborted  bool

	notifier Notifier
}

func newWorkerWatcher(req *neco.UpdateRequest, notifier Notifier) workerWatcher {
	watcher := workerWatcher{
		current:  req,
		statuses: make(map[int]*neco.UpdateStatus),
		aborted:  false,
		notifier: notifier,
	}
	return watcher
}

func (w *workerWatcher) handleStatus(ctx context.Context, lrn int, st *neco.UpdateStatus) (bool, error) {
	if st.Version != w.current.Version {
		return false, nil
	}
	w.statuses[lrn] = st

	switch st.Cond {
	case neco.CondAbort:
		log.Warn("worker failed updating", map[string]interface{}{
			"version": w.current.Version,
			"lrn":     lrn,
			"message": st.Message,
		})
		w.notifier.NotifyServerFailure(ctx, *w.current, st.Message)
		w.aborted = true
		return false, errors.New(st.Message)
	case neco.CondComplete:
		log.Info("worker finished updating", map[string]interface{}{
			"version": w.current.Version,
			"lrn":     lrn,
		})
	}

	success := neco.UpdateCompleted(w.current.Version, w.current.Servers, w.statuses)
	if success {
		log.Info("all worker finished updating", map[string]interface{}{
			"version": w.current.Version,
			"servers": w.current.Servers,
		})
		w.notifier.NotifySucceeded(ctx, *w.current)
		return true, nil
	}

	return false, nil
}

func (w *workerWatcher) handleError(ctx context.Context, err error) error {
	if err == storage.ErrTimedOut {
		log.Warn("workers timed-out", map[string]interface{}{
			"version":    w.current.Version,
			"started_at": w.current.StartedAt,
		})
		w.notifier.NotifyTimeout(ctx, *w.current)
	}
	return nil
}
