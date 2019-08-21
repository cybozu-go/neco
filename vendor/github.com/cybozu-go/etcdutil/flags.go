package etcdutil

import (
	"errors"
	"flag"
	"strings"

	"github.com/spf13/pflag"
)

type endpointsVal struct {
	*Config
}

func (v endpointsVal) String() string {
	if v.Config == nil {
		return "nil"
	}
	return strings.Join(v.Config.Endpoints, ",")
}

func (v endpointsVal) Set(s string) error {
	endpoints := strings.Split(s, ",")
	filtered := endpoints[:0]
	for _, e := range endpoints {
		if len(e) != 0 {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		return errors.New("no endpoints")
	}
	v.Config.Endpoints = filtered
	return nil
}

func (v endpointsVal) Type() string {
	return "endpoints"
}

// AddFlags adds common set of command-line flags for etcd.
func (c *Config) AddFlags(fs *flag.FlagSet) {
	fs.Var(endpointsVal{c}, "etcd-endpoints", "comma-separated list of URLs")
	fs.StringVar(&c.Prefix, "etcd-prefix", c.Prefix, "prefix for etcd keys")
	fs.StringVar(&c.Timeout, "etcd-timeout", c.Timeout, "dial timeout to etcd")
	fs.StringVar(&c.Username, "etcd-username", "", "username for etcd authentication")
	fs.StringVar(&c.Password, "etcd-password", "", "password for etcd authentication")
	fs.StringVar(&c.TLSCAFile, "etcd-tls-ca", "", "filename of etcd server TLS CA")
	fs.StringVar(&c.TLSCertFile, "etcd-tls-cert", "", "filename of etcd client certficate")
	fs.StringVar(&c.TLSKeyFile, "etcd-tls-key", "", "filename of etcd client private key")
}

// AddPFlags is a variation of AddFlags for github.com/spf13/pflag package.
func (c *Config) AddPFlags(fs *pflag.FlagSet) {
	fs.Var(endpointsVal{c}, "etcd-endpoints", "comma-separated list of URLs")
	fs.StringVar(&c.Prefix, "etcd-prefix", c.Prefix, "prefix for etcd keys")
	fs.StringVar(&c.Timeout, "etcd-timeout", c.Timeout, "dial timeout to etcd")
	fs.StringVar(&c.Username, "etcd-username", "", "username for etcd authentication")
	fs.StringVar(&c.Password, "etcd-password", "", "password for etcd authentication")
	fs.StringVar(&c.TLSCAFile, "etcd-tls-ca", "", "filename of etcd server TLS CA")
	fs.StringVar(&c.TLSCertFile, "etcd-tls-cert", "", "filename of etcd client certficate")
	fs.StringVar(&c.TLSKeyFile, "etcd-tls-key", "", "filename of etcd client private key")
}
