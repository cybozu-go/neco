module github.com/cybozu-go/cke

replace (
	labix.org/v2/mgo => github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	launchpad.net/gocheck => github.com/go-check/check v0.0.0-20180628173108-788fd7840127
)

require (
	github.com/99designs/gqlgen v0.9.3
	github.com/agnivade/levenshtein v1.0.2 // indirect
	github.com/containernetworking/cni v0.6.0
	github.com/coreos/etcd v3.3.19+incompatible
	github.com/cybozu-go/etcdutil v1.3.4
	github.com/cybozu-go/log v1.5.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/well v1.8.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a
	github.com/golang/groupcache v0.0.0-20181024230925-c65c006176ff // indirect
	github.com/google/go-cmp v0.3.0
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20200117231345-460d63e36490
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.4.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/vektah/gqlparser v1.1.2
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/apiserver v0.17.6
	k8s.io/client-go v0.17.6
	k8s.io/kube-scheduler v0.17.6
	k8s.io/kubelet v0.17.6
	sigs.k8s.io/yaml v1.2.0
)

go 1.13
