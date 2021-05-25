module github.com/cybozu-go/neco

go 1.16

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/cybozu-go/etcdutil v1.4.0
	github.com/cybozu-go/log v1.6.0
	github.com/cybozu-go/netutil v1.4.1
	github.com/cybozu-go/placemat/v2 v2.0.5
	github.com/cybozu-go/sabakan/v2 v2.6.0
	github.com/cybozu-go/well v1.10.0
	github.com/docker/docker v1.4.2-0.20191219165747-a9416c67da9f // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/go-containerregistry v0.5.0
	github.com/google/go-github/v35 v35.2.0
	github.com/hashicorp/go-msgpack v0.5.4 // indirect
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/serf v0.9.5
	github.com/hashicorp/vault/api v1.1.0
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/mattn/go-isatty v0.0.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6 // indirect
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.25.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stmcginnis/gofish v0.9.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/vektah/gqlparser v1.3.1
	github.com/vishvananda/netlink v1.1.0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20210512015243-d19fbe541bf9
	go.uber.org/multierr v1.5.0 // indirect
	go4.org v0.0.0-20190313082347-94abd6928b1d // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/oauth2 v0.0.0-20210427180440-81ed05c6b58c
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	sigs.k8s.io/yaml v1.2.0
)
