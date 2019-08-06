package k8s

import (
	"encoding/json"

	"github.com/cybozu-go/cke"
	"k8s.io/client-go/tools/clientcmd/api"
)

func controllerManagerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-controller-manager", ca, clientCrt, clientKey)
}

func schedulerKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-scheduler", ca, clientCrt, clientKey)
}

func proxyKubeconfig(cluster string, ca, clientCrt, clientKey string) *api.Config {
	return cke.Kubeconfig(cluster, "system:kube-proxy", ca, clientCrt, clientKey)
}

func kubeletKubeconfig(cluster string, n *cke.Node, caPath, certPath, keyPath string) *api.Config {
	cfg := api.NewConfig()
	c := api.NewCluster()
	c.Server = "https://localhost:16443"
	c.CertificateAuthority = caPath
	cfg.Clusters[cluster] = c

	auth := api.NewAuthInfo()
	auth.ClientCertificate = certPath
	auth.ClientKey = keyPath
	user := "system:node:" + n.Nodename()
	cfg.AuthInfos[user] = auth

	ctx := api.NewContext()
	ctx.AuthInfo = user
	ctx.Cluster = cluster
	cfg.Contexts["default"] = ctx
	cfg.CurrentContext = "default"

	return cfg
}

// KubeletConfiguration is a simplified version of the struct defined in
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/config/types.go
//
// Rationate: kubernetes repository is too large and not intended for client usage.
type KubeletConfiguration struct {
	APIVersion        string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind              string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Address           string `json:"address,omitempty" yaml:"address,omitempty"`
	Port              int32  `json:"port,omitempty" yaml:"port,omitempty"`
	ReadOnlyPort      int32  `json:"readOnlyPort" yaml:"readOnlyPort"`
	TLSCertFile       string `json:"tlsCertFile" yaml:"tlsCertFile"`
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile" yaml:"tlsPrivateKeyFile"`

	Authentication KubeletAuthentication `json:"authentication" yaml:"authentication"`
	Authorization  kubeletAuthorization  `json:"authorization" yaml:"authorization"`

	HealthzPort           int32    `json:"healthzPort,omitempty" yaml:"healthzPort,omitempty"`
	HealthzBindAddress    string   `json:"healthzBindAddress,omitempty" yaml:"healthzBindAddress,omitempty"`
	OOMScoreAdj           int32    `json:"oomScoreAdj" yaml:"oomScoreAdj"`
	ClusterDomain         string   `json:"clusterDomain,omitempty" yaml:"clusterDomain,omitempty"`
	ClusterDNS            []string `json:"clusterDNS,omitempty" yaml:"clusterDNS,omitempty"`
	PodCIDR               string   `json:"podCIDR,omitempty" yaml:"podCIDR,omitempty"`
	RuntimeRequestTimeout string   `json:"runtimeRequestTimeout,omitempty" yaml:"runtimeRequestTimeout,omitempty"`

	FeatureGates map[string]bool `json:"featureGates,omitempty" yaml:"featureGates,omitempty"`
	FailSwapOn   bool            `json:"failSwapOn" yaml:"failSwapOn"`

	ContainerLogMaxSize  string `json:"containerLogMaxSize,omitempty" yaml:"containerLogMaxSize,omitempty"`
	ContainerLogMaxFiles int32  `json:"containerLogMaxFiles,omitempty" yaml:"containerLogMaxFiles,omitempty"`
}

func newKubeletConfiguration(cert, key, ca, domain, logSize string, logFiles int32, allowSwap bool) KubeletConfiguration {
	return KubeletConfiguration{
		APIVersion:            "kubelet.config.k8s.io/v1beta1",
		Kind:                  "KubeletConfiguration",
		ReadOnlyPort:          0,
		TLSCertFile:           cert,
		TLSPrivateKeyFile:     key,
		Authentication:        KubeletAuthentication{ClientCAFile: ca},
		Authorization:         kubeletAuthorization{Mode: "Webhook"},
		HealthzBindAddress:    "0.0.0.0",
		OOMScoreAdj:           -1000,
		ClusterDomain:         domain,
		RuntimeRequestTimeout: "15m",
		FailSwapOn:            !allowSwap,
		ContainerLogMaxSize:   logSize,
		ContainerLogMaxFiles:  logFiles,
	}
}

// KubeletAuthentication is a simplified version of the struct defined in
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/config/types.go
//
// Rationate: kubernetes repository is too large and not intended for client usage.
type KubeletAuthentication struct {
	ClientCAFile string
}

// MarshalYAML implements yaml.Marshaler.
func (a KubeletAuthentication) MarshalYAML() (interface{}, error) {
	v := map[string]map[string]interface{}{}
	v["x509"] = map[string]interface{}{"clientCAFile": a.ClientCAFile}
	v["webhook"] = map[string]interface{}{"enabled": true}
	v["anonymous"] = map[string]interface{}{"enabled": false}
	return v, nil
}

// MarshalJSON implements json.Marshaler.
func (a KubeletAuthentication) MarshalJSON() ([]byte, error) {
	v, err := a.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// kubeletAuthorization is a simplified version of the struct defined in
// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/config/types.go
type kubeletAuthorization struct {
	Mode string `json:"mode" yaml:"mode"`
}

// EncryptionConfiguration is a simplified version of the struct defined in
// https://github.com/kubernetes/apiserver/blob/master/pkg/apis/config/types.go
//
// Rationate: kubernetes repository is too large and not intended for client usage.
type EncryptionConfiguration struct {
	APIVersion string                  `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string                  `json:"kind,omitempty" yaml:"kind,omitempty"`
	Resources  []ResourceConfiguration `json:"resources" yaml:"resources"`
}

func newEncryptionConfiguration() EncryptionConfiguration {
	return EncryptionConfiguration{
		APIVersion: "apiserver.config.k8s.io/v1",
		Kind:       "EncryptionConfiguration",
	}
}

// ResourceConfiguration is a simplified version of the struct defined in
// https://github.com/kubernetes/apiserver/blob/master/pkg/apis/config/types.go
type ResourceConfiguration struct {
	Resources []string                `json:"resources" yaml:"resources"`
	Providers []ProviderConfiguration `json:"providers" yaml:"providers"`
}

// ProviderConfiguration is a simplified version of the struct defined in
// https://github.com/kubernetes/apiserver/blob/master/pkg/apis/config/types.go
type ProviderConfiguration struct {
	AESCBC   *AESConfiguration `json:"aescbc,omitempty" yaml:"aescbc,omitempty"`
	Identity *struct{}         `json:"identity,omitempty" yaml:"identity,omitempty"`
}

// AESConfiguration is a simplified version of the struct defined in
// https://github.com/kubernetes/apiserver/blob/master/pkg/apis/config/types.go
type AESConfiguration struct {
	Keys []Key `json:"keys" yaml:"keys"`
}

// Key is a simplified version of the struct defined in
// https://github.com/kubernetes/apiserver/blob/master/pkg/apis/config/types.go
type Key struct {
	Name   string `json:"name" yaml:"name"`
	Secret string `json:"secret" yaml:"secret"`
}
