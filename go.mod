module github.com/cybozu-go/neco

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/containers/image/v5 v5.9.0
	github.com/coreos/etcd v3.3.19+incompatible
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/ignition v0.32.0 // indirect
	github.com/cybozu-go/etcdutil v1.3.4
	github.com/cybozu-go/log v1.6.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/placemat v1.5.3
	github.com/cybozu-go/sabakan/v2 v2.4.8
	github.com/cybozu-go/well v1.10.0
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/golang/protobuf v1.4.1 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github/v18 v18.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	github.com/hashicorp/go-msgpack v0.5.4 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/serf v0.8.3
	github.com/hashicorp/vault/api v1.0.5-0.20200117231345-460d63e36490
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/miekg/dns v1.1.8 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pelletier/go-toml v1.3.0 // indirect
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/prometheus/prom2json v1.1.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/tcnksm/go-input v0.0.0-20180404061846-548a7d7a8ee8
	github.com/vektah/gqlparser v1.1.2
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	go4.org v0.0.0-20190313082347-94abd6928b1d // indirect
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200513185701-a91f0712d120 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/tools v0.0.0-20200513201620-d5fe73897c97 // indirect
	google.golang.org/grpc v1.26.0 // indirect
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/golang/lint v0.0.0-20190409202823-959b441ac422 => golang.org/x/lint v0.0.0-20190409202823-959b441ac422

go 1.13
