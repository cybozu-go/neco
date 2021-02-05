module github.com/cybozu-go/neco

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/containers/image/v5 v5.9.0
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/ignition v0.32.0 // indirect
	github.com/cybozu-go/etcdutil v1.3.5
	github.com/cybozu-go/log v1.6.0
	github.com/cybozu-go/netutil v1.3.0
	github.com/cybozu-go/placemat v1.5.3
	github.com/cybozu-go/sabakan/v2 v2.5.3
	github.com/cybozu-go/well v1.10.0
	github.com/google/go-cmp v0.5.4
	github.com/google/go-containerregistry v0.4.0
	github.com/google/go-github/v33 v33.0.0
	github.com/hashicorp/go-msgpack v0.5.4 // indirect
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/serf v0.9.5
	github.com/hashicorp/vault/api v1.0.5-0.20210115204428-654c9ea2e306
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/mattn/go-isatty v0.0.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.2
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pelletier/go-toml v1.3.0 // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.15.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.1
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/vektah/gqlparser v1.3.1
	github.com/vishvananda/netlink v1.1.0
	go.uber.org/multierr v1.5.0 // indirect
	go4.org v0.0.0-20190313082347-94abd6928b1d // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	sigs.k8s.io/yaml v1.2.0
)

go 1.15
