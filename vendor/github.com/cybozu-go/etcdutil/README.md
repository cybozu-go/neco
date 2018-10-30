[![GitHub release](https://img.shields.io/github/release/cybozu-go/etcdutil.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/etcdutil.svg?style=svg)](https://circleci.com/gh/cybozu-go/etcdutil)
[![GoDoc](https://godoc.org/github.com/cybozu-go/etcdutil?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/etcdutil)](https://goreportcard.com/report/github.com/cybozu-go/etcdutil)

Add-ons for etcd
================

This package provides utility for using [etcd][].

Specifications
--------------

Programs using this package can implement following common specifications.

### YAML/JSON configuration file

The etcd parameters can be embedded in YAML or JSON file.

```yaml
endpoints:
  - http://10.1.2.3:2379
  - http://10.11.12.13:2379
prefix: "/key-prefix/"
timeout: "2s"

# user/pass authentication
username: "etcd-user-name"
password: "etcd-password"

# etcd server certificates' CA
tls-ca-file: "/etc/ssl/my-ca.crt"

# etcd client certificate authentication
tls-cert-file: "/path/to/my-user.crt"
tls-key-file: "/path/to/my-user.key"

# certificats may be embedded
tls-ca: |
  -----BEGIN CERTIFICATE-----
  MIICAzCCAWwCCQCgYvbe6d0oLzANBgkqhkiG9w0BAQUFADBGMQswCQYDVQQGEwJK
  ....
  -----END CERTIFICATE-----
tls-cert: |
  -----BEGIN CERTIFICATE-----
  MIICAzCCAWwCCQCgYvbe6d0oLzANBgkqhkiG9w0BAQUFADBGMQswCQYDVQQGEwJK
  ....
  -----END CERTIFICATE-----
tls-key: |
  -----BEGIN PRIVATE KEY-----
  MIICAzCCAWwCCQCgYvbe6d0oLzANBgkqhkiG9w0BAQUFADBGMQswCQYDVQQGEwJK
  ....
  -----END PRIVATE KEY-----
```

### Command-line flags

Name             | Default                 | Description
---------------- | ----------------------- | -----------
`etcd-endpoints` | `http://127.0.0.1:2379` | comma-separated URLs of the backend etcd
`etcd-password`  |                         | password for etcd authentication
`etcd-prefix`    |                         | prefix for etcd keys
`etcd-timeout`   | `2s`                    | dial timeout duration to etcd
`etcd-tls-ca`    |                         | Path to CA bundle used to verify etcd server certificates.
`etcd-tls-cert`  |                         | Path to client certificate file of an etcd user.
`etcd-tls-key`   |                         | Path to private key file of an etcd user.
`etcd-username`  |                         | username for etcd authentication

Usage
-----

Read [the documentation][godoc].

License
-------

etcdutil is licensed under MIT license.

[releases]: https://github.com/cybozu-go/etcdutil/releases
[godoc]: https://godoc.org/github.com/cybozu-go/etcdutil
[etcd]: https://coreos.com/etcd/
