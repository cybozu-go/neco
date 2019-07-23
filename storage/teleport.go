package storage

import (
	"context"
	"encoding/json"
)

// GetTeleportAuthServers returns auth servers.
func (s Storage) GetTeleportAuthServers(ctx context.Context) ([]string, error) {
	var servers []string
	serversJSON, err := s.get(ctx, KeyTeleportAuthServers)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(serversJSON), &servers)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// PutTeleportAuthServers stores auth servers' addresses
func (s Storage) PutTeleportAuthServers(ctx context.Context, servers []string) error {
	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return err
	}

	return s.put(ctx, KeyTeleportAuthServers, string(serversJSON))
}

// GetTeleportAuthToken returns auth token
func (s Storage) GetTeleportAuthToken(ctx context.Context) (string, error) {
	return s.get(ctx, KeyTeleportAuthToken)
}

// PutTeleportAuthToken stores auth token
func (s Storage) PutTeleportAuthToken(ctx context.Context, token string) error {
	return s.put(ctx, KeyTeleportAuthToken, token)
}
