package main

import "github.com/cybozu-go/etcdutil"

type config struct {
	HttpProxy string           `yaml:"http_proxy"`
	Etcd      *etcdutil.Config `yaml:"etcd"`
}
