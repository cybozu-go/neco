package storage

import (
	"context"
	"strconv"

	"github.com/coreos/etcd/clientv3"
)

// Finish stores finish flag for a bootserver to storage
func (s Storage) Finish(ctx context.Context, lrn int) error {
	_, err := s.etcd.Put(ctx, keyFinish(lrn), "")
	return err
}

// GetFinished returns list of bootservers that completed setup
func (s Storage) GetFinished(ctx context.Context) ([]int, error) {
	resp, err := s.etcd.Get(ctx, KeyFinishPrefix,
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, nil
	}
	lrns := make([]int, resp.Count)
	for i, kv := range resp.Kvs {
		lrn, err := strconv.Atoi(string(kv.Key[len(KeyFinishPrefix):]))
		if err != nil {
			return nil, err
		}
		lrns[i] = lrn
	}
	return lrns, nil
}
