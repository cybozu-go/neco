package cke

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
)

// Storage provides operations to store/retrieve CKE data in etcd.
type Storage struct {
	*clientv3.Client
}

// RecordChan is a channel for watching new operation records.
type RecordChan <-chan *Record

// etcd keys and prefixes
const (
	KeyCA                    = "ca/"
	KeyCluster               = "cluster"
	KeyClusterRevision       = "cluster-revision"
	KeyConstraints           = "constraints"
	KeyLeader                = "leader/"
	KeyRecords               = "records/"
	KeyRecordID              = "records"
	KeyResourcePrefix        = "resource/"
	KeySabakanDisabled       = "sabakan/disabled"
	KeySabakanQueryVariables = "sabakan/query-variables"
	KeySabakanTemplate       = "sabakan/template"
	KeySabakanURL            = "sabakan/url"
	KeyServiceAccountCert    = "service-account/certificate"
	KeyServiceAccountKey     = "service-account/key"
	KeyVault                 = "vault"
)

const maxRecords = 1000
const recordChanLength = 100
const initialDisplayCount = 20

var (
	// ErrNotFound may be returned by Storage methods when a key is not found.
	ErrNotFound = errors.New("not found")
	// ErrNoLeader is returned when the session lost leadership.
	ErrNoLeader = errors.New("lost leadership")
)

// PutCluster stores *Cluster into etcd.
func (s Storage) PutCluster(ctx context.Context, c *Cluster) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeyCluster, string(data))
	return err
}

// PutClusterWithTemplateRevision stores *Cluster into etcd along with a revision number.
func (s Storage) PutClusterWithTemplateRevision(ctx context.Context, c *Cluster, rev int64, leaderKey string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(KeyCluster, string(data)),
			clientv3.OpPut(KeyClusterRevision, strconv.FormatInt(rev, 10)),
		).Commit()
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// GetCluster loads *Cluster from etcd.
// If cluster configuration has not been stored, this returns ErrNotFound.
func (s Storage) GetCluster(ctx context.Context) (*Cluster, error) {
	resp, err := s.Get(ctx, KeyCluster)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	c := new(Cluster)
	err = json.Unmarshal(resp.Kvs[0].Value, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetClusterWithRevision loads *Cluster from etcd as well as the stored
// revision number.  The revision number was stored with *Cluster by
// PutClusterWithTemplateRevision().
func (s Storage) GetClusterWithRevision(ctx context.Context) (*Cluster, int64, error) {
	resp, err := s.Txn(ctx).
		Then(
			clientv3.OpGet(KeyCluster),
			clientv3.OpGet(KeyClusterRevision),
		).Commit()
	if err != nil {
		return nil, 0, err
	}
	if !resp.Succeeded {
		panic("transaction without if condition failed")
	}

	gresp0 := resp.Responses[0].GetResponseRange()
	gresp1 := resp.Responses[1].GetResponseRange()
	if len(gresp0.Kvs) == 0 {
		return nil, 0, ErrNotFound
	}

	c := new(Cluster)
	err = json.Unmarshal(gresp0.Kvs[0].Value, c)
	if err != nil {
		return nil, 0, err
	}

	var rev int64
	if len(gresp1.Kvs) > 0 {
		rev, err = strconv.ParseInt(string(gresp1.Kvs[0].Value), 10, 64)
		if err != nil {
			return nil, 0, err
		}
	}

	return c, rev, nil
}

// PutConstraints stores *Constraints into etcd.
func (s Storage) PutConstraints(ctx context.Context, c *Constraints) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeyConstraints, string(data))
	return err
}

// GetConstraints loads *Constraints from etcd.
// If constraints have not been stored, this returns ErrNotFound.
func (s Storage) GetConstraints(ctx context.Context) (*Constraints, error) {
	resp, err := s.Get(ctx, KeyConstraints)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	c := new(Constraints)
	err = json.Unmarshal(resp.Kvs[0].Value, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// PutVaultConfig stores *VaultConfig into etcd.
func (s Storage) PutVaultConfig(ctx context.Context, c *VaultConfig) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeyVault, string(data))
	return err
}

// GetVaultConfig loads *VaultConfig from etcd.
func (s Storage) GetVaultConfig(ctx context.Context) (*VaultConfig, error) {
	resp, err := s.Get(ctx, KeyVault)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	cfg := new(VaultConfig)
	err = json.Unmarshal(resp.Kvs[0].Value, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetCACertificate loads CA certificate from etcd.
func (s Storage) GetCACertificate(ctx context.Context, name string) (string, error) {
	resp, err := s.Get(ctx, KeyCA+name)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// PutCACertificate stores CA certificate into etcd.
func (s Storage) PutCACertificate(ctx context.Context, name, pem string) error {
	_, err := s.Put(ctx, KeyCA+name, pem)
	return err
}

func recordKey(r *Record) string {
	return fmt.Sprintf("%s%016x", KeyRecords, r.ID)
}

// GetServiceAccountCert loads x509 certificate for service account.
// The format is PEM.
func (s Storage) GetServiceAccountCert(ctx context.Context) (string, error) {
	resp, err := s.Get(ctx, KeyServiceAccountCert)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// GetServiceAccountKey loads private key for service account.
// The format is PEM.
func (s Storage) GetServiceAccountKey(ctx context.Context) (string, error) {
	resp, err := s.Get(ctx, KeyServiceAccountKey)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// PutServiceAccountData stores x509 certificate and private key for service account.
func (s Storage) PutServiceAccountData(ctx context.Context, leaderKey, cert, key string) error {
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(KeyServiceAccountCert, cert),
			clientv3.OpPut(KeyServiceAccountKey, key)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// GetRecords loads list of *Record from etcd.
// The returned records are sorted by record ID in decreasing order.
func (s Storage) GetRecords(ctx context.Context, count int64) ([]*Record, error) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend),
	}
	if count > 0 {
		opts = append(opts, clientv3.WithLimit(count))
	}
	resp, err := s.Get(ctx, KeyRecords, opts...)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	records := make([]*Record, len(resp.Kvs))

	for i, kv := range resp.Kvs {
		r := new(Record)
		err = json.Unmarshal(kv.Value, r)
		if err != nil {
			return nil, err
		}
		records[i] = r
	}

	return records, nil
}

// WatchRecords watches new operation records.
// The watched records will be returned through the returned channel.
func (s Storage) WatchRecords(ctx context.Context, initialCount int64) (RecordChan, error) {
	if initialCount == 0 {
		initialCount = initialDisplayCount
	}

	getOpts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend),
		clientv3.WithLimit(initialCount),
	}
	getResp, err := s.Get(ctx, KeyRecords, getOpts...)
	if err != nil {
		return nil, err
	}

	getRecords := make([]*Record, len(getResp.Kvs))
	for i, kv := range getResp.Kvs {
		r := new(Record)
		err = json.Unmarshal(kv.Value, r)
		if err != nil {
			return nil, err
		}
		getRecords[i] = r
	}
	sort.SliceStable(getRecords, func(i, j int) bool { return getRecords[i].ID < getRecords[j].ID })

	watchOpt := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithRev(getResp.Header.Revision + 1),
		clientv3.WithFilterDelete(),
	}
	watchCh := s.Watch(ctx, KeyRecords, watchOpt...)

	recordCh := make(chan *Record, recordChanLength)

	go func() {
		defer func() {
			close(recordCh)
		}()

		for _, r := range getRecords {
			recordCh <- r
		}

		for watchResp := range watchCh {
			err := watchResp.Err()
			if err != nil {
				return
			}

			for _, ev := range watchResp.Events {
				r := new(Record)
				err := json.Unmarshal(ev.Kv.Value, r)
				if err != nil {
					return
				}
				recordCh <- r
			}
		}
		return
	}()

	return recordCh, nil
}

// RegisterRecord stores *Record if the leaderKey exists
func (s Storage) RegisterRecord(ctx context.Context, leaderKey string, r *Record) error {
	nextID := strconv.FormatInt(r.ID+1, 10)
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(recordKey(r), string(data)),
			clientv3.OpPut(KeyRecordID, nextID)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}

	return s.maintRecords(ctx, leaderKey, maxRecords)
}

// UpdateRecord updates existing record
func (s Storage) UpdateRecord(ctx context.Context, leaderKey string, r *Record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpPut(recordKey(r), string(data))).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// NextRecordID get the next record ID from etcd
func (s Storage) NextRecordID(ctx context.Context) (int64, error) {
	resp, err := s.Get(ctx, KeyRecordID)
	if err != nil {
		return 0, err
	}
	if len(resp.Kvs) == 0 {
		return 1, nil
	}

	id, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s Storage) maintRecords(ctx context.Context, leaderKey string, max int64) error {
	resp, err := s.Get(ctx, KeyRecords,
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return err
	}

	if len(resp.Kvs) <= int(max) {
		return nil
	}

	startKey := string(resp.Kvs[0].Key)
	endKey := string(resp.Kvs[len(resp.Kvs)-int(max)].Key)

	tresp, err := s.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpDelete(startKey, clientv3.WithRange(endKey))).
		Commit()
	if !tresp.Succeeded {
		return ErrNoLeader
	}
	return err
}

// GetLeaderHostname returns the current leader's host name.
// It returns non-nil error when there is no leader.
func (s Storage) GetLeaderHostname(ctx context.Context) (string, error) {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	opts = append(opts, clientv3.WithFirstCreate()...)
	resp, err := s.Get(ctx, KeyLeader, opts...)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", errors.New("no leader")
	}
	return string(resp.Kvs[0].Value), nil
}

// ListResources lists keys of registered user resources.
func (s Storage) ListResources(ctx context.Context) ([]string, error) {
	resp, err := s.Get(ctx, KeyResourcePrefix,
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	keys := make([]string, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		keys[i] = string(kv.Key[len(KeyResourcePrefix):])
	}
	return keys, nil
}

// GetResource gets a user resource.
func (s Storage) GetResource(ctx context.Context, key string) ([]byte, int64, error) {
	resp, err := s.Get(ctx, KeyResourcePrefix+key)
	if err != nil {
		return nil, 0, err
	}

	if len(resp.Kvs) == 0 {
		return nil, 0, ErrNotFound
	}

	return resp.Kvs[0].Value, resp.Kvs[0].ModRevision, nil
}

// GetAllResources gets all user-defined resources.
// The returned slice of resources are sorted so that creating resources in order
// will not fail.
func (s Storage) GetAllResources(ctx context.Context) ([]ResourceDefinition, error) {
	resp, err := s.Get(ctx, KeyResourcePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	rcs := make([]ResourceDefinition, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		key := string(kv.Key[len(KeyResourcePrefix):])
		parts := strings.Split(key, "/")
		kind := Kind(parts[0])

		if !kind.IsSupported() {
			// ignore unsupported resources
			continue
		}

		var namespace, name string
		switch len(parts) {
		case 2:
			name = parts[1]
		case 3:
			namespace = parts[1]
			name = parts[2]
		default:
			return nil, errors.New("invalid resource key: " + key)
		}

		rcs = append(rcs, ResourceDefinition{
			Key:        key,
			Kind:       kind,
			Namespace:  namespace,
			Name:       name,
			Revision:   kv.ModRevision,
			Definition: kv.Value,
		})
	}

	SortResources(rcs)
	return rcs, nil
}

// SetResource sets a user resource.
func (s Storage) SetResource(ctx context.Context, key, value string) error {
	_, err := s.Put(ctx, KeyResourcePrefix+key, value)
	return err
}

// DeleteResource removes a user resource from etcd.
func (s Storage) DeleteResource(ctx context.Context, key string) error {
	_, err := s.Delete(ctx, KeyResourcePrefix+key)
	return err
}

// IsSabakanDisabled returns true if sabakan integration is disabled.
func (s Storage) IsSabakanDisabled(ctx context.Context) (bool, error) {
	resp, err := s.Get(ctx, KeySabakanDisabled)
	if err != nil {
		return false, err
	}
	if resp.Count == 0 {
		return false, nil
	}

	if bytes.Equal([]byte("true"), resp.Kvs[0].Value) {
		return true, nil
	}
	return false, nil
}

// EnableSabakan enables sabakan integration when flag is true.
// When flag is false, sabakan integration is disabled.
func (s Storage) EnableSabakan(ctx context.Context, flag bool) error {
	val := fmt.Sprint(!flag)
	_, err := s.Put(ctx, KeySabakanDisabled, val)
	return err
}

// SetSabakanQueryVariables sets query variables for Sabakan.
// Caller must validate the contents.
func (s Storage) SetSabakanQueryVariables(ctx context.Context, vars string) error {
	_, err := s.Put(ctx, KeySabakanQueryVariables, vars)
	return err
}

// GetSabakanQueryVariables gets query variables for Sabakan.
func (s Storage) GetSabakanQueryVariables(ctx context.Context) ([]byte, error) {
	resp, err := s.Get(ctx, KeySabakanQueryVariables)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	return resp.Kvs[0].Value, nil
}

// SetSabakanTemplate stores template cluster configuration.
// Caller must validate the template.
func (s Storage) SetSabakanTemplate(ctx context.Context, tmpl *Cluster) error {
	data, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}

	_, err = s.Put(ctx, KeySabakanTemplate, string(data))
	return err
}

// GetSabakanTemplate gets template cluster configuration.
// If a template exists, it will be returned with ModRevision.
func (s Storage) GetSabakanTemplate(ctx context.Context) (*Cluster, int64, error) {
	resp, err := s.Get(ctx, KeySabakanTemplate)
	if err != nil {
		return nil, 0, err
	}

	if len(resp.Kvs) == 0 {
		return nil, 0, ErrNotFound
	}

	tmpl := new(Cluster)
	err = json.Unmarshal(resp.Kvs[0].Value, tmpl)
	if err != nil {
		return nil, 0, err
	}

	return tmpl, resp.Kvs[0].ModRevision, nil
}

// SetSabakanURL stores URL of sabakan API.
func (s Storage) SetSabakanURL(ctx context.Context, url string) error {
	_, err := s.Put(ctx, KeySabakanURL, url)
	return err
}

// GetSabakanURL gets URL of sabakan API.
// The URL must be an absolute URL pointing GraphQL endpoint.
func (s Storage) GetSabakanURL(ctx context.Context) (string, error) {
	resp, err := s.Get(ctx, KeySabakanURL)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}
