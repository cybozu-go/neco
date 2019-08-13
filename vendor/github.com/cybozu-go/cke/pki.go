package cke

import (
	"context"
	"net"
	"path"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/netutil"
	vault "github.com/hashicorp/vault/api"
)

// CNAPIServer is the common name of API server for aggregation
const CNAPIServer = "front-proxy-client"

// CA keys in Vault
const (
	CAServer                = "cke/ca-server"
	CAEtcdPeer              = "cke/ca-etcd-peer"
	CAEtcdClient            = "cke/ca-etcd-client"
	CAKubernetes            = "cke/ca-kubernetes"
	CAKubernetesAggregation = "cke/ca-kubernetes-aggregation"
)

// CAKeys is list of CA keys
var CAKeys = []string{
	CAServer,
	CAEtcdPeer,
	CAEtcdClient,
	CAKubernetes,
	CAKubernetesAggregation,
}

// Role name in Vault
const (
	RoleSystem                = "system"
	RoleAdmin                 = "admin"
	RoleKubeScheduler         = "kube-scheduler"
	RoleKubeControllerManager = "kube-controller-manager"
	RoleKubelet               = "kubelet"
	RoleKubeProxy             = "kube-proxy"
	RoleServiceAccount        = "service-account"
)

// IssueResponse is cli output format.
type IssueResponse struct {
	Cert   string `json:"certificate"`
	Key    string `json:"private_key"`
	CACert string `json:"ca_certificate"`
}

// addRole adds a role to CA if not exists.
func addRole(client *vault.Client, ca, role string, data map[string]interface{}) error {
	l := client.Logical()
	rpath := path.Join(ca, "roles", role)
	secret, err := l.Read(rpath)
	if err != nil {
		return err
	}
	if secret != nil {
		// already exists
		return nil
	}

	_, err = l.Write(rpath, data)
	if err != nil {
		log.Error("failed to create vault role", map[string]interface{}{
			log.FnError: err,
			"ca":        ca,
			"role":      role,
		})
	}
	return err
}

// EtcdCA is a certificate authority for etcd cluster.
type EtcdCA struct{}

// IssueServerCert issues TLS server certificates.
func (e EtcdCA) IssueServerCert(ctx context.Context, inf Infrastructure, node *Node, domain string) (crt, key string, err error) {
	altNames := []string{
		"localhost",
		"cke-etcd",
		"cke-etcd.kube-system",
		"cke-etcd.kube-system.svc",
	}
	d := strings.Split(domain, ".")
	for i := range d {
		altNames = append(altNames, "cke-etcd.kube-system.svc."+strings.Join(d[0:i+1], "."))
	}
	return issueCertificate(inf, CAServer, RoleSystem,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"client_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name": node.Nodename(),
			"alt_names":   strings.Join(altNames, ","),
			"ip_sans":     "127.0.0.1," + node.Address,
		})
}

// IssuePeerCert issues TLS certificates for mutual peer authentication.
func (e EtcdCA) IssuePeerCert(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAEtcdPeer, RoleSystem,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          node.Nodename(),
			"ip_sans":              "127.0.0.1," + node.Address,
			"exclude_cn_from_sans": "true",
		})
}

// IssueForAPIServer issues TLC client certificate for Kubernetes.
func (e EtcdCA) IssueForAPIServer(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, RoleSystem,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          "kube-apiserver",
			"exclude_cn_from_sans": "true",
		})
}

// IssueRoot issues certificate for root user.
func (e EtcdCA) IssueRoot(ctx context.Context, inf Infrastructure) (cert, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, RoleAdmin,
		map[string]interface{}{
			"ttl":            "2h",
			"max_ttl":        "24h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          "root",
			"exclude_cn_from_sans": "true",
			"ttl":                  "1h",
		})
}

// IssueForBackup issues certificate for etcdbackup.
func (e EtcdCA) IssueForBackup(ctx context.Context, inf Infrastructure) (cert, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, RoleAdmin,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          "root",
			"exclude_cn_from_sans": "true",
		})
}

// IssueEtcdClientCertificate issues TLS client certificate for a user.
func IssueEtcdClientCertificate(inf Infrastructure, username, ttl string) (cert, key string, err error) {
	return issueCertificate(inf, CAEtcdClient, RoleSystem,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          username,
			"exclude_cn_from_sans": "true",
			"ttl":                  ttl,
		})
}

// KubernetesCA is a certificate authority for k8s cluster.
type KubernetesCA struct{}

// IssueAdminCert issues client certificate for cluster admin user.
func (k KubernetesCA) IssueAdminCert(ctx context.Context, inf Infrastructure, ttl string) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, RoleAdmin,
		map[string]interface{}{
			"ttl":               "2h",
			"max_ttl":           "48h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:masters",
		},
		map[string]interface{}{
			"ttl":                  ttl,
			"common_name":          "admin",
			"exclude_cn_from_sans": "true",
		})
}

// IssueForAPIServer issues TLS certificate for API servers.
func (k KubernetesCA) IssueForAPIServer(ctx context.Context, inf Infrastructure, n *Node, serviceSubnet, domain string) (crt, key string, err error) {
	altNames := []string{
		"localhost",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
	}
	d := strings.Split(domain, ".")
	for i := range d {
		altNames = append(altNames, "kubernetes.default.svc."+strings.Join(d[0:i+1], "."))
	}
	ip, _, err := net.ParseCIDR(serviceSubnet)
	if err != nil {
		return "", "", err
	}
	kubeSvcAddr := netutil.IntToIP4(netutil.IP4ToInt(ip) + 1)

	return issueCertificate(inf, CAKubernetes, RoleSystem,
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
		},
		map[string]interface{}{
			"common_name":          "kubernetes",
			"alt_names":            strings.Join(altNames, ","),
			"ip_sans":              "127.0.0.1," + n.Address + "," + kubeSvcAddr.String(),
			"exclude_cn_from_sans": "true",
		})
}

// IssueForScheduler issues TLS certificate for kube-scheduler.
func (k KubernetesCA) IssueForScheduler(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, RoleKubeScheduler,
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:kube-scheduler",
		},
		map[string]interface{}{
			"common_name":          "system:kube-scheduler",
			"exclude_cn_from_sans": "true",
		})
}

// IssueForControllerManager issues TLS certificate for kube-controller-manager.
func (k KubernetesCA) IssueForControllerManager(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, RoleKubeControllerManager,
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:kube-controller-manager",
		},
		map[string]interface{}{
			"common_name":          "system:kube-controller-manager",
			"exclude_cn_from_sans": "true",
		})
}

// IssueForKubelet issues TLS certificate for kubelet.
func (k KubernetesCA) IssueForKubelet(ctx context.Context, inf Infrastructure, node *Node) (crt, key string, err error) {
	nodename := node.Nodename()
	altNames := "localhost"
	if nodename != node.Address {
		altNames = "localhost," + nodename
	}

	return issueCertificate(inf, CAKubernetes, RoleKubelet,
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:nodes",
		},
		map[string]interface{}{
			"common_name":          "system:node:" + nodename,
			"alt_names":            altNames,
			"ip_sans":              "127.0.0.1," + node.Address,
			"exclude_cn_from_sans": "true",
		})
}

// IssueForProxy issues TLS certificate for kube-proxy.
func (k KubernetesCA) IssueForProxy(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, RoleKubeProxy,
		map[string]interface{}{
			"ttl":               "87600h",
			"max_ttl":           "87600h",
			"enforce_hostnames": "false",
			"allow_any_name":    "true",
			"organization":      "system:node-proxier",
		},
		map[string]interface{}{
			"common_name":          "system:kube-proxy",
			"exclude_cn_from_sans": "true",
		})
}

// IssueForServiceAccount issues TLS certificate to sign service account tokens.
func (k KubernetesCA) IssueForServiceAccount(ctx context.Context, inf Infrastructure) (crt, key string, err error) {
	return issueCertificate(inf, CAKubernetes, RoleServiceAccount,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"allow_any_name": "true",
			"client_flag":    "false",
			"server_flag":    "false",
			"key_usage":      "DigitalSignature,CertSign",
			"no_store":       "true",
		},
		map[string]interface{}{
			"common_name":          "service-account",
			"exclude_cn_from_sans": "true",
		})
}

// AggregationCA is a certificate authority for kubernetes aggregation API server
type AggregationCA struct{}

// IssueClientCertificate issues TLS client certificate for API server
func (a AggregationCA) IssueClientCertificate(ctx context.Context, inf Infrastructure) (cert, key string, err error) {
	return issueCertificate(inf, CAKubernetesAggregation, RoleSystem,
		map[string]interface{}{
			"ttl":            "87600h",
			"max_ttl":        "87600h",
			"server_flag":    "false",
			"allow_any_name": "true",
		},
		map[string]interface{}{
			"common_name":          CNAPIServer,
			"exclude_cn_from_sans": "true",
		})
}

func issueCertificate(inf Infrastructure, ca, role string, roleOpts, certOpts map[string]interface{}) (crt, key string, err error) {
	client, err := inf.Vault()
	if err != nil {
		return "", "", err
	}

	err = addRole(client, ca, role, roleOpts)
	if err != nil {
		return "", "", err
	}

	secret, err := client.Logical().Write(path.Join(ca, "issue", role), certOpts)
	if err != nil {
		return "", "", err
	}
	crt = secret.Data["certificate"].(string)
	key = secret.Data["private_key"].(string)
	return crt, key, err
}
