package op

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/scheduler"
	"github.com/cybozu-go/cke/static"
	"github.com/cybozu-go/log"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// GetNodeStatus returns NodeStatus.
func GetNodeStatus(ctx context.Context, inf cke.Infrastructure, node *cke.Node, cluster *cke.Cluster) (*cke.NodeStatus, error) {
	status := &cke.NodeStatus{}
	agent := inf.Agent(node.Address)
	status.SSHConnected = agent != nil
	if !status.SSHConnected {
		return status, nil
	}

	ce := inf.Engine(node.Address)
	ss, err := ce.Inspect([]string{
		EtcdContainerName,
		RiversContainerName,
		EtcdRiversContainerName,
		KubeAPIServerContainerName,
		KubeControllerManagerContainerName,
		KubeSchedulerContainerName,
		KubeProxyContainerName,
		KubeletContainerName,
	})
	if err != nil {
		return nil, err
	}

	etcdVolumeExists, err := ce.VolumeExists(EtcdVolumeName(cluster.Options.Etcd))
	if err != nil {
		return nil, err
	}

	status.Etcd = cke.EtcdStatus{
		ServiceStatus: ss[EtcdContainerName],
		HasData:       etcdVolumeExists,
	}
	status.Rivers = ss[RiversContainerName]
	status.EtcdRivers = ss[EtcdRiversContainerName]

	status.APIServer = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeAPIServerContainerName],
		IsHealthy:     false,
	}
	if status.APIServer.Running {
		status.APIServer.IsHealthy, err = checkAPIServerHealth(ctx, inf, node)
		if err != nil {
			log.Warn("failed to check API server health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.ControllerManager = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeControllerManagerContainerName],
		IsHealthy:     false,
	}
	if status.ControllerManager.Running {
		status.ControllerManager.IsHealthy, err = checkSecureHealthz(ctx, inf, node.Address, 10257)
		if err != nil {
			log.Warn("failed to check controller manager health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}
	}

	status.Scheduler = cke.SchedulerStatus{
		ServiceStatus: ss[KubeSchedulerContainerName],
		IsHealthy:     false,
	}

	if status.Scheduler.Running {
		status.Scheduler.IsHealthy, err = checkSecureHealthz(ctx, inf, node.Address, 10259)
		if err != nil {
			log.Warn("failed to check scheduler health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}

		var policy scheduler.Policy
		// Testing policy file existence is needed for backward compatibility
		policyStr, _, err := agent.Run(fmt.Sprintf("if [ -f %s ]; then cat %s; fi",
			PolicyConfigPath, PolicyConfigPath))
		if err != nil {
			log.Error("failed to cat "+PolicyConfigPath, map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
			return nil, err
		}
		err = yaml.Unmarshal(policyStr, &policy)
		if err != nil {
			log.Error("failed to unmarshal policy config json", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
				"data":      policyStr,
			})
			return nil, err
		}
		status.Scheduler.Extenders = policy.ExtenderConfigs
	}

	// TODO: due to the following bug, health status cannot be checked for proxy.
	// https://github.com/kubernetes/kubernetes/issues/65118
	status.Proxy = cke.KubeComponentStatus{
		ServiceStatus: ss[KubeProxyContainerName],
		IsHealthy:     false,
	}
	status.Proxy.IsHealthy = status.Proxy.Running

	status.Kubelet = cke.KubeletStatus{
		ServiceStatus: ss[KubeletContainerName],
		IsHealthy:     false,
		Domain:        "",
		AllowSwap:     false,
	}
	if status.Kubelet.Running {
		status.Kubelet.IsHealthy, err = CheckKubeletHealthz(ctx, inf, node.Address, 10248)
		if err != nil {
			log.Warn("failed to check kubelet health", map[string]interface{}{
				log.FnError: err,
				"node":      node.Address,
			})
		}

		cfgData, _, err := agent.Run("cat /etc/kubernetes/kubelet/config.yml")
		if err == nil {
			v := struct {
				ClusterDomain        string `json:"clusterDomain"`
				FailSwapOn           bool   `json:"failSwapOn"`
				ContainerLogMaxSize  string `json:"containerLogMaxSize"`
				ContainerLogMaxFiles int32  `json:"containerLogMaxFiles"`
			}{}
			err = yaml.Unmarshal(cfgData, &v)
			if err == nil {
				status.Kubelet.Domain = v.ClusterDomain
				status.Kubelet.AllowSwap = !v.FailSwapOn
				status.Kubelet.ContainerLogMaxSize = v.ContainerLogMaxSize
				status.Kubelet.ContainerLogMaxFiles = v.ContainerLogMaxFiles
			}
		}
	}

	return status, nil
}

// GetEtcdClusterStatus returns EtcdClusterStatus
func GetEtcdClusterStatus(ctx context.Context, inf cke.Infrastructure, nodes []*cke.Node) (cke.EtcdClusterStatus, error) {
	clusterStatus := cke.EtcdClusterStatus{}

	var endpoints []string
	for _, n := range nodes {
		if n.ControlPlane {
			endpoints = append(endpoints, fmt.Sprintf("https://%s:2379", n.Address))
		}
	}

	cli, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return clusterStatus, err
	}
	defer cli.Close()

	clusterStatus.Members, err = getEtcdMembers(ctx, inf, cli)
	if err != nil {
		return clusterStatus, err
	}

	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
	defer cancel()
	resp, err := cli.Grant(ct, 10)
	if err != nil {
		return clusterStatus, err
	}

	clusterStatus.IsHealthy = resp.ID != clientv3.NoLease

	clusterStatus.InSyncMembers = make(map[string]bool)
	for name := range clusterStatus.Members {
		clusterStatus.InSyncMembers[name] = getEtcdMemberInSync(ctx, inf, name, resp.Revision)
	}

	return clusterStatus, nil
}

func getEtcdMembers(ctx context.Context, inf cke.Infrastructure, cli *clientv3.Client) (map[string]*etcdserverpb.Member, error) {
	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return nil, err
	}
	members := make(map[string]*etcdserverpb.Member)
	for _, m := range resp.Members {
		name, err := GuessMemberName(m)
		if err != nil {
			return nil, err
		}
		members[name] = m
	}
	return members, nil
}

// GuessMemberName returns etcd member's ip address
func GuessMemberName(m *etcdserverpb.Member) (string, error) {
	if len(m.Name) > 0 {
		return m.Name, nil
	}

	if len(m.PeerURLs) == 0 {
		return "", errors.New("empty PeerURLs")
	}

	u, err := url.Parse(m.PeerURLs[0])
	if err != nil {
		return "", err
	}
	h, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return h, nil
}

func getEtcdMemberInSync(ctx context.Context, inf cke.Infrastructure, address string, clusterRev int64) bool {
	endpoints := []string{fmt.Sprintf("https://%s:2379", address)}
	cli, err := inf.NewEtcdClient(ctx, endpoints)
	if err != nil {
		return false
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, TimeoutDuration)
	defer cancel()
	resp, err := cli.Get(ct, "health")
	if err != nil {
		return false
	}

	return resp.Header.Revision >= clusterRev
}

// GetKubernetesClusterStatus returns KubernetesClusterStatus
func GetKubernetesClusterStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node, cluster *cke.Cluster) (cke.KubernetesClusterStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s := cke.KubernetesClusterStatus{}

	_, err = clientset.CoreV1().ServiceAccounts("kube-system").Get("default", metav1.GetOptions{})
	switch {
	case err == nil:
		s.IsControlPlaneReady = true
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	resp, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	s.Nodes = resp.Items

	if len(cluster.DNSService) > 0 {
		fields := strings.Split(cluster.DNSService, "/")
		if len(fields) != 2 {
			panic("invalid dns_service in cluster.yml")
		}
		svc, err := clientset.CoreV1().Services(fields[0]).Get(fields[1], metav1.GetOptions{})
		switch {
		case k8serr.IsNotFound(err):
		case err == nil:
			s.DNSService = svc
		default:
			return cke.KubernetesClusterStatus{}, err
		}
	}

	s.ClusterDNS, err = getClusterDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	s.NodeDNS, err = getNodeDNSStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	epAPI := clientset.CoreV1().Endpoints
	ep, err := epAPI(metav1.NamespaceDefault).Get("kubernetes", metav1.GetOptions{})
	switch {
	case err == nil:
		s.MasterEndpoints = ep
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	svc, err := clientset.CoreV1().Services(metav1.NamespaceSystem).Get(EtcdServiceName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.EtcdService = svc
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	ep, err = epAPI(metav1.NamespaceSystem).Get(EtcdEndpointsName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.EtcdEndpoints = ep
	case k8serr.IsNotFound(err):
	default:
		return cke.KubernetesClusterStatus{}, err
	}

	s.EtcdBackup, err = getEtcdBackupStatus(ctx, inf, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}

	rkeys, err := inf.Storage().ListResources(ctx)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	for _, res := range static.Resources {
		rkeys = append(rkeys, res.Key)
	}

	k8s, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.KubernetesClusterStatus{}, err
	}
	s.ResourceStatuses = make(map[string]map[string]string)
	for _, rkey := range rkeys {
		parts := strings.Split(rkey, "/")
		switch parts[0] {
		case "Namespace":
			obj, err := k8s.CoreV1().Namespaces().Get(parts[1], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "ServiceAccount":
			obj, err := k8s.CoreV1().ServiceAccounts(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "ConfigMap":
			obj, err := k8s.CoreV1().ConfigMaps(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "Service":
			obj, err := k8s.CoreV1().Services(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "PodSecurityPolicy":
			obj, err := k8s.PolicyV1beta1().PodSecurityPolicies().Get(parts[1], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "NetworkPolicy":
			obj, err := k8s.NetworkingV1().NetworkPolicies(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "Role":
			obj, err := k8s.RbacV1().Roles(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "RoleBinding":
			obj, err := k8s.RbacV1().RoleBindings(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "ClusterRole":
			obj, err := k8s.RbacV1().ClusterRoles().Get(parts[1], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "ClusterRoleBinding":
			obj, err := k8s.RbacV1().ClusterRoleBindings().Get(parts[1], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "Deployment":
			obj, err := k8s.AppsV1().Deployments(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "DaemonSet":
			obj, err := k8s.AppsV1().DaemonSets(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "CronJob":
			obj, err := k8s.BatchV2alpha1().CronJobs(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		case "PodDisruptionBudget":
			obj, err := k8s.PolicyV1beta1().PodDisruptionBudgets(parts[1]).Get(parts[2], metav1.GetOptions{})
			if k8serr.IsNotFound(err) {
				continue
			}
			if err != nil {
				return cke.KubernetesClusterStatus{}, err
			}
			s.SetResourceStatus(rkey, obj.Annotations)
		default:
			log.Warn("unknown resource kind", map[string]interface{}{
				"kind": parts[0],
			})
		}
	}

	return s, nil
}

func getClusterDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.ClusterDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.ClusterDNSStatus{}, err
	}

	s := cke.ClusterDNSStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ClusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case k8serr.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	service, err := clientset.CoreV1().Services("kube-system").Get(ClusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ClusterIP = service.Spec.ClusterIP
	case k8serr.IsNotFound(err):
	default:
		return cke.ClusterDNSStatus{}, err
	}

	return s, nil
}

func getNodeDNSStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.NodeDNSStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.NodeDNSStatus{}, err
	}

	s := cke.NodeDNSStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(NodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case k8serr.IsNotFound(err):
	default:
		return cke.NodeDNSStatus{}, err
	}

	return s, nil
}

func getEtcdBackupStatus(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (cke.EtcdBackupStatus, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return cke.EtcdBackupStatus{}, err
	}

	s := cke.EtcdBackupStatus{}

	config, err := clientset.CoreV1().ConfigMaps("kube-system").Get(EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.ConfigMap = config
	case k8serr.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	pod, err := clientset.CoreV1().Pods("kube-system").Get(EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Pod = pod
	case k8serr.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	service, err := clientset.CoreV1().Services("kube-system").Get(EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Service = service
	case k8serr.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	secret, err := clientset.CoreV1().Secrets("kube-system").Get(EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.Secret = secret
	case k8serr.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	job, err := clientset.BatchV1beta1().CronJobs("kube-system").Get(EtcdBackupAppName, metav1.GetOptions{})
	switch {
	case err == nil:
		s.CronJob = job
	case k8serr.IsNotFound(err):
	default:
		return cke.EtcdBackupStatus{}, err
	}

	return s, nil
}

// CheckKubeletHealthz checks that Kubelet is healthy
func CheckKubeletHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
	healthzURL := "http://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", healthzURL, nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)
	resp, err := inf.HTTPClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(body)) == "ok", nil
}

func checkSecureHealthz(ctx context.Context, inf cke.Infrastructure, addr string, port uint16) (bool, error) {
	healthzURL := "https://" + addr + ":" + strconv.FormatUint(uint64(port), 10) + "/healthz"
	req, err := http.NewRequest("GET", healthzURL, nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)
	client, err := inf.HTTPSClient(ctx)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(body)) == "ok", nil
}

func checkAPIServerHealth(ctx context.Context, inf cke.Infrastructure, n *cke.Node) (bool, error) {
	clientset, err := inf.K8sClient(ctx, n)
	if err != nil {
		return false, err
	}
	_, err = clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func containCommandOption(slice []string, optionName string) bool {
	for _, v := range slice {
		switch {
		case v == optionName:
			return true
		case strings.HasPrefix(v, optionName+"="):
			return true
		case strings.HasPrefix(v, optionName+" "):
			return true
		}
	}
	return false
}
