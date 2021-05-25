package storage

import (
	"context"
	"strconv"

	"go.etcd.io/etcd/clientv3"
)

// Finish stores the finished stage number for a bootserver to storage
func (s Storage) Finish(ctx context.Context, lrn int, stage int) error {
	_, err := s.etcd.Put(ctx, keyFinish(lrn), strconv.Itoa(stage))
	return err
}

// GetFinished returns a list of bootservers that completed specified stage of setup
func (s Storage) GetFinished(ctx context.Context, stage int) ([]int, error) {
	resp, err := s.etcd.Get(ctx, KeyFinishPrefix,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, nil
	}
	var lrns []int
	for _, kv := range resp.Kvs {
		current, err := strconv.Atoi(string(kv.Value))
		if err != nil {
			return nil, err
		}
		if current != stage {
			continue
		}
		lrn, err := strconv.Atoi(string(kv.Key[len(KeyFinishPrefix):]))
		if err != nil {
			return nil, err
		}
		lrns = append(lrns, lrn)
	}
	return lrns, nil
}
