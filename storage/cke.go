package storage

import (
	"context"
	"encoding/json"
)

// PutCKEWeight stores weights of roles.
func (s Storage) PutCKEWeight(ctx context.Context, value map[string]float64) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.put(ctx, KeyCKEWeight, string(data))
}

// GetCKEWeight returns weights of roles..
func (s Storage) GetCKEWeight(ctx context.Context) (map[string]float64, error) {
	resp, err := s.etcd.Get(ctx, KeyCKEWeight)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	var rw map[string]float64
	err = json.Unmarshal(resp.Kvs[0].Value, &rw)
	if err != nil {
		return nil, err
	}

	return rw, nil
}
