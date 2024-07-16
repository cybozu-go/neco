package necorebooter

import (
	"io"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/etcdutil"
	"github.com/robfig/cron/v3"
	"sigs.k8s.io/yaml"
)

const DefaultMetricsPort = 10082

type Config struct {
	RebootTimes   []RebootTimes `json:"rebootTimes"`
	TimeZone      string        `json:"timeZone"`
	GroupLabelKey string        `json:"groupLabelKey"`
	MetricsPort   int           `json:"metricsPort"`
}

type RebootTimes struct {
	Name          string        `json:"name"`
	LabelSelector LabelSelector `json:"labelSelector"`
	Times         Times         `json:"times"`
}

type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels"`
}

type Times struct {
	Deny  []string `json:"deny"`
	Allow []string `json:"allow"`
}

type RebootTime struct {
	Deny  []cron.Schedule
	Allow []cron.Schedule
}

func LoadConfig(reader io.Reader) (*Config, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	config := &Config{
		MetricsPort: DefaultMetricsPort,
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) GetRebootTime() (map[string]RebootTime, error) {
	rebootTime := map[string]RebootTime{}
	for _, rt := range c.RebootTimes {
		deny := []cron.Schedule{}
		allow := []cron.Schedule{}
		for _, d := range rt.Times.Deny {
			s, err := cron.ParseStandard(d)
			if err != nil {
				return nil, err
			}
			deny = append(deny, s)
		}
		for _, a := range rt.Times.Allow {
			s, err := cron.ParseStandard(a)
			if err != nil {
				return nil, err
			}
			allow = append(allow, s)
		}
		rebootTime[rt.Name] = RebootTime{
			Deny:  deny,
			Allow: allow,
		}
	}
	return rebootTime, nil
}

func NewCKEStorage(ckeConfig io.Reader) (*cke.Storage, error) {
	b, err := io.ReadAll(ckeConfig)
	if err != nil {
		return nil, err
	}
	cfg := cke.NewEtcdConfig()
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}
	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	storage := cke.Storage{Client: etcd}
	return &storage, nil
}
