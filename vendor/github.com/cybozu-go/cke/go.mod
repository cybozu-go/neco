module github.com/cybozu-go/cke

replace (
	labix.org/v2/mgo => github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	launchpad.net/gocheck => github.com/go-check/check v0.0.0-20180628173108-788fd7840127
)

require (
	github.com/99designs/gqlgen v0.7.1
	github.com/agnivade/levenshtein v1.0.1 // indirect
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883 // indirect
	github.com/containernetworking/cni v0.6.0
	github.com/coreos/bbolt v1.3.1-coreos.6 // indirect
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/cybozu-go/etcdutil v1.3.1
	github.com/cybozu-go/log v1.5.0
	github.com/cybozu-go/netutil v1.2.0
	github.com/cybozu-go/well v1.8.1
	github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.2.0
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/hashicorp/vault/api v1.0.2
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/onsi/ginkgo v1.6.1-0.20180913204007-46bbc26e8a91
	github.com/onsi/gomega v1.4.2
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.0
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/vektah/gqlparser v1.0.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.1
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.0.0-20190313235455-40a48860b5ab
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.1.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190306001800-15615b16d372 // indirect
	k8s.io/utils v0.0.0-20190308190857-21c4ce38f2a7 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
