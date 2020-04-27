package cke

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containernetworking/cni/libcni"
	corev1 "k8s.io/api/core/v1"
	v1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"
	"sigs.k8s.io/yaml"
)

// Node represents a node in Kubernetes.
type Node struct {
	Address      string            `json:"address"`
	Hostname     string            `json:"hostname"`
	User         string            `json:"user"`
	ControlPlane bool              `json:"control_plane"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	Taints       []corev1.Taint    `json:"taints"`
}

// Nodename returns a hostname or address if hostname is empty
func (n *Node) Nodename() string {
	if len(n.Hostname) == 0 {
		return n.Address
	}
	return n.Hostname
}

// BindPropagation is bind propagation option for Docker
// https://docs.docker.com/storage/bind-mounts/#configure-bind-propagation
type BindPropagation string

// Bind propagation definitions
const (
	PropagationShared   = BindPropagation("shared")
	PropagationSlave    = BindPropagation("slave")
	PropagationPrivate  = BindPropagation("private")
	PropagationRShared  = BindPropagation("rshared")
	PropagationRSlave   = BindPropagation("rslave")
	PropagationRPrivate = BindPropagation("rprivate")
)

func (p BindPropagation) String() string {
	return string(p)
}

// SELinuxLabel is selinux label of the host file or directory
// https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label
type SELinuxLabel string

// SELinux Label definitions
const (
	LabelShared  = SELinuxLabel("z")
	LabelPrivate = SELinuxLabel("Z")
)

func (l SELinuxLabel) String() string {
	return string(l)
}

// Mount is volume mount information
type Mount struct {
	Source      string          `json:"source"`
	Destination string          `json:"destination"`
	ReadOnly    bool            `json:"read_only"`
	Propagation BindPropagation `json:"propagation"`
	Label       SELinuxLabel    `json:"selinux_label"`
}

// Equal returns true if the mount is equals to other one, otherwise return false
func (m Mount) Equal(o Mount) bool {
	return m.Source == o.Source && m.Destination == o.Destination && m.ReadOnly == o.ReadOnly
}

// ServiceParams is a common set of extra parameters for k8s components.
type ServiceParams struct {
	ExtraArguments []string          `json:"extra_args"`
	ExtraBinds     []Mount           `json:"extra_binds"`
	ExtraEnvvar    map[string]string `json:"extra_env"`
}

// Equal returns true if the services params is equals to other one, otherwise return false
func (s ServiceParams) Equal(o ServiceParams) bool {
	return compareStrings(s.ExtraArguments, o.ExtraArguments) &&
		compareMounts(s.ExtraBinds, o.ExtraBinds) &&
		compareStringMap(s.ExtraEnvvar, o.ExtraEnvvar)
}

// EtcdParams is a set of extra parameters for etcd.
type EtcdParams struct {
	ServiceParams `json:",inline"`
	VolumeName    string `json:"volume_name"`
}

// APIServerParams is a set of extra parameters for kube-apiserver.
type APIServerParams struct {
	ServiceParams   `json:",inline"`
	AuditLogEnabled bool   `json:"audit_log_enabled"`
	AuditLogPolicy  string `json:"audit_log_policy"`
}

// CNIConfFile is a config file for CNI plugin deployed on worker nodes by CKE.
type CNIConfFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// SchedulerParams is a set of extra parameters for kube-scheduler.
type SchedulerParams struct {
	ServiceParams `json:",inline"`
	Extenders     []string `json:"extenders"`
}

// KubeletParams is a set of extra parameters for kubelet.
type KubeletParams struct {
	ServiceParams            `json:",inline"`
	ContainerRuntime         string         `json:"container_runtime"`
	ContainerRuntimeEndpoint string         `json:"container_runtime_endpoint"`
	ContainerLogMaxSize      string         `json:"container_log_max_size"`
	ContainerLogMaxFiles     int32          `json:"container_log_max_files"`
	Domain                   string         `json:"domain"`
	AllowSwap                bool           `json:"allow_swap"`
	BootTaints               []corev1.Taint `json:"boot_taints"`
	CNIConfFile              CNIConfFile    `json:"cni_conf_file"`
}

// EtcdBackup is a set of configurations for etcdbackup.
type EtcdBackup struct {
	Enabled  bool   `json:"enabled"`
	PVCName  string `json:"pvc_name"`
	Schedule string `json:"schedule"`
	Rotate   int    `json:"rotate,omitempty"`
}

// Options is a set of optional parameters for k8s components.
type Options struct {
	Etcd              EtcdParams      `json:"etcd"`
	Rivers            ServiceParams   `json:"rivers"`
	EtcdRivers        ServiceParams   `json:"etcd-rivers"`
	APIServer         APIServerParams `json:"kube-api"`
	ControllerManager ServiceParams   `json:"kube-controller-manager"`
	Scheduler         SchedulerParams `json:"kube-scheduler"`
	Proxy             ServiceParams   `json:"kube-proxy"`
	Kubelet           KubeletParams   `json:"kubelet"`
}

// Cluster is a set of configurations for a etcd/Kubernetes cluster.
type Cluster struct {
	Name          string     `json:"name"`
	Nodes         []*Node    `json:"nodes"`
	TaintCP       bool       `json:"taint_control_plane"`
	ServiceSubnet string     `json:"service_subnet"`
	PodSubnet     string     `json:"pod_subnet"`
	DNSServers    []string   `json:"dns_servers"`
	DNSService    string     `json:"dns_service"`
	EtcdBackup    EtcdBackup `json:"etcd_backup"`
	Options       Options    `json:"options"`
}

// Validate validates the cluster definition.
func (c *Cluster) Validate(isTmpl bool) error {
	if len(c.Name) == 0 {
		return errors.New("cluster name is empty")
	}

	_, _, err := net.ParseCIDR(c.ServiceSubnet)
	if err != nil {
		return err
	}
	_, _, err = net.ParseCIDR(c.PodSubnet)
	if err != nil {
		return err
	}

	fldPath := field.NewPath("nodes")
	for i, n := range c.Nodes {
		err := c.validateNode(n, isTmpl, fldPath.Index(i))
		if err != nil {
			return err
		}
	}

	for _, a := range c.DNSServers {
		if net.ParseIP(a) == nil {
			return errors.New("invalid IP address: " + a)
		}
	}

	if len(c.DNSService) > 0 {
		fields := strings.Split(c.DNSService, "/")
		if len(fields) != 2 {
			return errors.New("invalid DNS service (no namespace?): " + c.DNSService)
		}
	}

	err = validateEtcdBackup(c.EtcdBackup)
	if err != nil {
		return err
	}

	err = validateOptions(c.Options)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) validateNode(n *Node, isTmpl bool, fldPath *field.Path) error {
	if !isTmpl {
		if net.ParseIP(n.Address) == nil {
			return errors.New("invalid IP address: " + n.Address)
		}
	}
	if len(n.User) == 0 {
		return errors.New("user name is empty")
	}

	if err := validateNodeLabels(n, fldPath.Child("labels")); err != nil {
		return err
	}
	if err := validateNodeAnnotations(n, fldPath.Child("annotations")); err != nil {
		return err
	}
	if err := validateNodeTaints(n, fldPath.Child("taints")); err != nil {
		return err
	}
	return nil
}

// validateNodeLabels validates label names and values with
// rules described in:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func validateNodeLabels(n *Node, fldPath *field.Path) error {
	el := v1validation.ValidateLabels(n.Labels, fldPath)
	if len(el) == 0 {
		return nil
	}
	return el.ToAggregate()
}

// validateNodeAnnotations validates annotation names.
// The validation logic references:
// https://github.com/kubernetes/apimachinery/blob/60666be32c5de527b69dabe8e4400b4f0aa897de/pkg/api/validation/objectmeta.go#L50
func validateNodeAnnotations(n *Node, fldPath *field.Path) error {
	for k := range n.Annotations {
		msgs := validation.IsQualifiedName(strings.ToLower(k))
		if len(msgs) > 0 {
			el := make(field.ErrorList, len(msgs))
			for i, msg := range msgs {
				el[i] = field.Invalid(fldPath, k, msg)
			}
			return el.ToAggregate()
		}
	}
	return nil
}

// validateNodeTaints validates taint names, values, and effects.
func validateNodeTaints(n *Node, fldPath *field.Path) error {
	for i, taint := range n.Taints {
		err := validateTaint(taint, fldPath.Index(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// validateTaint validates a taint name, value, and effect.
// The validation logic references:
// https://github.com/kubernetes/kubernetes/blob/7cbb9995189c5ecc8182da29cd0e30188c911401/pkg/apis/core/validation/validation.go#L4105
func validateTaint(taint corev1.Taint, fldPath *field.Path) error {
	el := v1validation.ValidateLabelName(taint.Key, fldPath.Child("key"))
	if msgs := validation.IsValidLabelValue(taint.Value); len(msgs) > 0 {
		el = append(el, field.Invalid(fldPath.Child("value"), taint.Value, strings.Join(msgs, ";")))
	}
	switch taint.Effect {
	case corev1.TaintEffectNoSchedule:
	case corev1.TaintEffectPreferNoSchedule:
	case corev1.TaintEffectNoExecute:
	default:
		el = append(el, field.Invalid(fldPath.Child("effect"), string(taint.Effect), "invalid effect"))
	}
	if len(el) > 0 {
		return el.ToAggregate()
	}
	return nil
}

// ControlPlanes returns control plane []*Node
func ControlPlanes(nodes []*Node) []*Node {
	return filterNodes(nodes, func(n *Node) bool {
		return n.ControlPlane
	})
}

func filterNodes(nodes []*Node, f func(n *Node) bool) []*Node {
	var filtered []*Node
	for _, n := range nodes {
		if f(n) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func validateEtcdBackup(etcdBackup EtcdBackup) error {
	if etcdBackup.Enabled == false {
		return nil
	}
	if len(etcdBackup.PVCName) == 0 {
		return errors.New("pvc_name is empty")
	}
	if len(etcdBackup.Schedule) == 0 {
		return errors.New("schedule is empty")
	}
	return nil
}

func validateOptions(opts Options) error {
	v := func(binds []Mount) error {
		for _, m := range binds {
			if !filepath.IsAbs(m.Source) {
				return errors.New("source path must be absolute: " + m.Source)
			}
			if !filepath.IsAbs(m.Destination) {
				return errors.New("destination path must be absolute: " + m.Destination)
			}
		}
		return nil
	}

	err := v(opts.Etcd.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.APIServer.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.ControllerManager.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Scheduler.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Proxy.ExtraBinds)
	if err != nil {
		return err
	}
	err = v(opts.Kubelet.ExtraBinds)
	if err != nil {
		return err
	}

	fldPath := field.NewPath("options", "kubelet")
	if len(opts.Kubelet.Domain) > 0 {
		msgs := validation.IsDNS1123Subdomain(opts.Kubelet.Domain)
		if len(msgs) > 0 {
			return field.Invalid(fldPath.Child("domain"),
				opts.Kubelet.Domain, strings.Join(msgs, ";"))
		}
	}
	if len(opts.Kubelet.ContainerRuntime) > 0 {
		if opts.Kubelet.ContainerRuntime != "remote" && opts.Kubelet.ContainerRuntime != "docker" {
			return errors.New("kubelet.container_runtime should be 'docker' or 'remote'")
		}
		if opts.Kubelet.ContainerRuntime == "remote" && len(opts.Kubelet.ContainerRuntimeEndpoint) == 0 {
			return errors.New("kubelet.container_runtime_endpoint should not be empty")
		}
	}
	if len(opts.Kubelet.CNIConfFile.Content) != 0 && len(opts.Kubelet.CNIConfFile.Name) == 0 {
		return fmt.Errorf("kubelet.cni_conf_file.name should not be empty when kubelet.cni_conf_file.content is not empty")
	}
	if filename := opts.Kubelet.CNIConfFile.Name; len(filename) != 0 {
		matched, err := regexp.Match(`^[0-9A-Za-z_.-]+$`, []byte(filename))
		if err != nil {
			return err
		}
		if !matched {
			return errors.New(filename + " is invalid as file name")
		}

		if filepath.Ext(opts.Kubelet.CNIConfFile.Name) == ".conflist" {
			_, err = libcni.ConfListFromBytes([]byte(opts.Kubelet.CNIConfFile.Content))
			if err != nil {
				return err
			}
		} else {
			_, err = libcni.ConfFromBytes([]byte(opts.Kubelet.CNIConfFile.Content))
			if err != nil {
				return err
			}
		}
	}

	fldPath = fldPath.Child("boot_taints")
	for i, taint := range opts.Kubelet.BootTaints {
		err := validateTaint(taint, fldPath.Index(i))
		if err != nil {
			return err
		}
	}

	if opts.APIServer.AuditLogEnabled && len(opts.APIServer.AuditLogPolicy) == 0 {
		return errors.New("audit_log_policy should not be empty")
	}

	if len(opts.APIServer.AuditLogPolicy) != 0 {
		policy := make(map[string]interface{})
		err = yaml.Unmarshal([]byte(opts.APIServer.AuditLogPolicy), &policy)
		if err != nil {
			return err
		}
	}

	for _, e := range opts.Scheduler.Extenders {
		config := schedulerv1.Extender{}
		err = yaml.Unmarshal([]byte(e), &config)
		if err != nil {
			return err
		}
		if len(config.URLPrefix) == 0 {
			return errors.New("no urlPrefix is provided")
		}
		if _, err = url.Parse(config.URLPrefix); err != nil {
			return err
		}
	}

	return nil
}
