package main

import "github.com/cybozu-go/etcdutil"

type config struct {
	HTTPProxy string           `yaml:"http_proxy"`
	Etcd      *etcdutil.Config `yaml:"etcd"`
}
