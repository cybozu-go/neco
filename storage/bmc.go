package storage

import "context"

// PutBMCBMCUser stores bmc-user.json contents
func (s Storage) PutBMCBMCUser(ctx context.Context, value string) error {
	return s.put(ctx, KeyBMCBMCUser, value)
}

// GetBMCBMCUser returns bmc-user.json contents
func (s Storage) GetBMCBMCUser(ctx context.Context) (string, error) {
	return s.get(ctx, KeyBMCBMCUser)
}

// PutBMCIPMIUser stores IPMI username.
func (s Storage) PutBMCIPMIUser(ctx context.Context, value string) error {
	return s.put(ctx, KeyBMCIPMIUser, value)
}

// GetBMCIPMIUser returns IPMI username.
func (s Storage) GetBMCIPMIUser(ctx context.Context) (string, error) {
	return s.get(ctx, KeyBMCIPMIUser)
}

// PutBMCIPMIPassword stores IPMI password.
func (s Storage) PutBMCIPMIPassword(ctx context.Context, value string) error {
	return s.put(ctx, KeyBMCIPMIPassword, value)
}

// GetBMCIPMIPassword returns IPMI password.
func (s Storage) GetBMCIPMIPassword(ctx context.Context) (string, error) {
	return s.get(ctx, KeyBMCIPMIPassword)
}
