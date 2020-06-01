package k8s

import (
	"bytes"
	"time"

	"github.com/cybozu-go/cke"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	apiserverv1 "k8s.io/apiserver/pkg/apis/config/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
)

var (
	resourceEncoder runtime.Encoder
	scm             = runtime.NewScheme()
)

func init() {
	if err := apiserverv1.AddToScheme(scm); err != nil {
		panic(err)
	}
	if err := kubeletv1beta1.AddToScheme(scm); err != nil {
		panic(err)
	}
	resourceEncoder = json.NewSerializerWithOptions(json.DefaultMetaFactory, scm, scm, json.SerializerOptions{Yaml: true})
}

func encodeToYAML(obj runtime.Object) ([]byte, error) {
	unst := &unstructured.Unstructured{}
	if err := scm.Convert(obj, unst, nil); err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := resourceEncoder.Encode(unst, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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

func newKubeletConfiguration(cert, key, ca string, params cke.KubeletParams) kubeletv1beta1.KubeletConfiguration {
	return kubeletv1beta1.KubeletConfiguration{
		ReadOnlyPort:      0,
		TLSCertFile:       cert,
		TLSPrivateKeyFile: key,
		Authentication: kubeletv1beta1.KubeletAuthentication{
			X509:    kubeletv1beta1.KubeletX509Authentication{ClientCAFile: ca},
			Webhook: kubeletv1beta1.KubeletWebhookAuthentication{Enabled: boolPointer(true)},
		},
		Authorization:         kubeletv1beta1.KubeletAuthorization{Mode: kubeletv1beta1.KubeletAuthorizationModeWebhook},
		HealthzBindAddress:    "0.0.0.0",
		OOMScoreAdj:           int32Pointer(-1000),
		ClusterDomain:         params.Domain,
		RuntimeRequestTimeout: metav1.Duration{Duration: 15 * time.Minute},
		FailSwapOn:            boolPointer(!params.AllowSwap),
		CgroupDriver:          params.CgroupDriver,
		ContainerLogMaxSize:   params.ContainerLogMaxSize,
		ContainerLogMaxFiles:  int32Pointer(params.ContainerLogMaxFiles),
	}
}

func int32Pointer(input int32) *int32 {
	return &input
}

func boolPointer(input bool) *bool {
	return &input
}
