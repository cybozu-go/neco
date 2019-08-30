package cke

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

// Annotations for CKE-managed resources.
const (
	AnnotationResourceImage    = "cke.cybozu.com/image"
	AnnotationResourceRevision = "cke.cybozu.com/revision"
	AnnotationResourceOriginal = "cke.cybozu.com/last-applied-configuration"
)

// Kind prepresents Kubernetes resource kind
type Kind string

// Supported resource kinds
const (
	KindNamespace          = "Namespace"
	KindServiceAccount     = "ServiceAccount"
	KindPodSecurityPolicy  = "PodSecurityPolicy"
	KindNetworkPolicy      = "NetworkPolicy"
	KindClusterRole        = "ClusterRole"
	KindRole               = "Role"
	KindClusterRoleBinding = "ClusterRoleBinding"
	KindRoleBinding        = "RoleBinding"
	KindConfigMap          = "ConfigMap"
	KindDeployment         = "Deployment"
	KindDaemonSet          = "DaemonSet"
	KindCronJob            = "CronJob"
	KindService            = "Service"
)

// IsSupported returns true if k is supported by CKE.
func (k Kind) IsSupported() bool {
	switch k {
	case KindNamespace, KindServiceAccount,
		KindPodSecurityPolicy, KindNetworkPolicy,
		KindClusterRole, KindRole, KindClusterRoleBinding, KindRoleBinding,
		KindConfigMap, KindDeployment, KindDaemonSet, KindCronJob, KindService:
		return true
	}
	return false
}

// Order returns the precedence of resource creation order as an integer.
func (k Kind) Order() int {
	switch k {
	case KindNamespace:
		return 1
	case KindServiceAccount:
		return 2
	case KindPodSecurityPolicy:
		return 3
	case KindNetworkPolicy:
		return 4
	case KindClusterRole:
		return 5
	case KindRole:
		return 6
	case KindClusterRoleBinding:
		return 7
	case KindRoleBinding:
		return 8
	case KindConfigMap:
		return 9
	case KindDeployment:
		return 10
	case KindDaemonSet:
		return 11
	case KindCronJob:
		return 12
	case KindService:
		return 13
	}
	panic("unknown kind: " + string(k))
}

var resourceDecoder runtime.Decoder
var resourceEncoder runtime.Encoder

func init() {
	gvs := runtime.GroupVersioners{
		runtime.NewMultiGroupVersioner(corev1.SchemeGroupVersion),
		runtime.NewMultiGroupVersioner(policyv1beta1.SchemeGroupVersion),
		runtime.NewMultiGroupVersioner(networkingv1.SchemeGroupVersion),
		runtime.NewMultiGroupVersioner(rbacv1.SchemeGroupVersion),
		runtime.NewMultiGroupVersioner(appsv1.SchemeGroupVersion),
		runtime.NewMultiGroupVersioner(batchv1beta1.SchemeGroupVersion),
	}
	resourceDecoder = scheme.Codecs.DecoderToVersion(scheme.Codecs.UniversalDeserializer(), gvs)
	resourceEncoder = json.NewSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, false)
}

func encodeToJSON(obj runtime.Object) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := resourceEncoder.Encode(obj, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ApplyResource creates or patches Kubernetes object.
func ApplyResource(clientset *kubernetes.Clientset, data []byte, rev int64) error {
	obj, gvk, err := resourceDecoder.Decode(data, nil, nil)
	if err != nil {
		return err
	}

	switch o := obj.(type) {
	case *corev1.Namespace:
		c := clientset.CoreV1().Namespaces()
		return applyNamespace(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *corev1.ServiceAccount:
		c := clientset.CoreV1().ServiceAccounts(o.Namespace)
		return applyServiceAccount(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *corev1.ConfigMap:
		c := clientset.CoreV1().ConfigMaps(o.Namespace)
		return applyConfigMap(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *corev1.Service:
		c := clientset.CoreV1().Services(o.Namespace)
		return applyService(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *policyv1beta1.PodSecurityPolicy:
		c := clientset.PolicyV1beta1().PodSecurityPolicies()
		return applyPodSecurityPolicy(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *networkingv1.NetworkPolicy:
		c := clientset.NetworkingV1().NetworkPolicies(o.Namespace)
		return applyNetworkPolicy(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *rbacv1.Role:
		c := clientset.RbacV1().Roles(o.Namespace)
		return applyRole(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *rbacv1.RoleBinding:
		c := clientset.RbacV1().RoleBindings(o.Namespace)
		return applyRoleBinding(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *rbacv1.ClusterRole:
		c := clientset.RbacV1().ClusterRoles()
		return applyClusterRole(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *rbacv1.ClusterRoleBinding:
		c := clientset.RbacV1().ClusterRoleBindings()
		return applyClusterRoleBinding(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *appsv1.Deployment:
		c := clientset.AppsV1().Deployments(o.Namespace)
		return applyDeployment(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *appsv1.DaemonSet:
		c := clientset.AppsV1().DaemonSets(o.Namespace)
		return applyDaemonSet(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	case *batchv1beta1.CronJob:
		c := clientset.BatchV1beta1().CronJobs(o.Namespace)
		return applyCronJob(o, data, rev, c.Get, c.Create, c.Patch, c.Delete)
	}

	return fmt.Errorf("unsupported type: %s", gvk.String())
}

// ParseResource parses YAML string.
func ParseResource(data []byte) (key string, jsonData []byte, err error) {
	obj, gvk, err := resourceDecoder.Decode(data, nil, nil)
	if err != nil {
		return "", nil, err
	}

	switch o := obj.(type) {
	case *corev1.Namespace:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Name, data, err
	case *corev1.ServiceAccount:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *corev1.ConfigMap:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *corev1.Service:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *policyv1beta1.PodSecurityPolicy:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Name, data, err
	case *networkingv1.NetworkPolicy:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.Role:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.RoleBinding:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.ClusterRole:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Name, data, err
	case *rbacv1.ClusterRoleBinding:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Name, data, err
	case *appsv1.Deployment:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *appsv1.DaemonSet:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *batchv1beta1.CronJob:
		data, err := encodeToJSON(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	}

	return "", nil, fmt.Errorf("unsupported type: %s", gvk.String())
}

// ResourceDefinition represents a CKE-managed kubernetes resource.
type ResourceDefinition struct {
	Key        string
	Kind       Kind
	Namespace  string
	Name       string
	Revision   int64
	Image      string
	Definition []byte
}

// String implements fmt.Stringer.
func (d ResourceDefinition) String() string {
	return fmt.Sprintf("%s@%d", d.Key, d.Revision)
}

// NeedUpdate returns true if annotations of the current resource
// indicates need for update.
func (d ResourceDefinition) NeedUpdate(annotations map[string]string) bool {
	curRev, ok := annotations[AnnotationResourceRevision]
	if !ok {
		return true
	}
	if curRev != strconv.FormatInt(d.Revision, 10) {
		return true
	}

	if d.Image == "" {
		return false
	}

	curImage, ok := annotations[AnnotationResourceImage]
	if !ok {
		return true
	}
	return curImage != d.Image
}

// SortResources sort resources as defined order of creation.
func SortResources(res []ResourceDefinition) {
	less := func(i, j int) bool {
		a := res[i]
		b := res[j]
		if a.Kind != b.Kind {
			return a.Kind.Order() < b.Kind.Order()
		}
		switch i := strings.Compare(a.Namespace, b.Namespace); i {
		case -1:
			return true
		case 1:
			return false
		}
		switch i := strings.Compare(a.Name, b.Name); i {
		case -1:
			return true
		case 1:
			return false
		}
		// equal
		return false
	}

	sort.Slice(res, less)
}
