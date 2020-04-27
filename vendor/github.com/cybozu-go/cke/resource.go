package cke

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cybozu-go/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
)

// Annotations for CKE-managed resources.
const (
	AnnotationResourceImage    = "cke.cybozu.com/image"
	AnnotationResourceRevision = "cke.cybozu.com/revision"
)

// Kind represents Kubernetes resource kind
type Kind string

// Supported resource kinds
const (
	KindNamespace           = "Namespace"
	KindServiceAccount      = "ServiceAccount"
	KindPodSecurityPolicy   = "PodSecurityPolicy"
	KindNetworkPolicy       = "NetworkPolicy"
	KindClusterRole         = "ClusterRole"
	KindRole                = "Role"
	KindClusterRoleBinding  = "ClusterRoleBinding"
	KindRoleBinding         = "RoleBinding"
	KindConfigMap           = "ConfigMap"
	KindDeployment          = "Deployment"
	KindDaemonSet           = "DaemonSet"
	KindCronJob             = "CronJob"
	KindService             = "Service"
	KindPodDisruptionBudget = "PodDisruptionBudget"
)

// IsSupported returns true if k is supported by CKE.
func (k Kind) IsSupported() bool {
	switch k {
	case KindNamespace, KindServiceAccount,
		KindPodSecurityPolicy, KindNetworkPolicy,
		KindClusterRole, KindRole, KindClusterRoleBinding, KindRoleBinding,
		KindConfigMap, KindDeployment, KindDaemonSet, KindCronJob, KindService, KindPodDisruptionBudget:
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
	case KindPodDisruptionBudget:
		return 14
	}
	panic("unknown kind: " + string(k))
}

var resourceDecoder runtime.Decoder

func init() {
	gvs := schema.GroupVersions{
		schema.GroupVersion{Group: corev1.SchemeGroupVersion.Group, Version: corev1.SchemeGroupVersion.Version},
		schema.GroupVersion{Group: policyv1beta1.SchemeGroupVersion.Group, Version: policyv1beta1.SchemeGroupVersion.Version},
		schema.GroupVersion{Group: networkingv1.SchemeGroupVersion.Group, Version: networkingv1.SchemeGroupVersion.Version},
		schema.GroupVersion{Group: rbacv1.SchemeGroupVersion.Group, Version: rbacv1.SchemeGroupVersion.Version},
		schema.GroupVersion{Group: appsv1.SchemeGroupVersion.Group, Version: appsv1.SchemeGroupVersion.Version},
		schema.GroupVersion{Group: batchv1beta1.SchemeGroupVersion.Group, Version: batchv1beta1.SchemeGroupVersion.Version},
	}
	resourceDecoder = scheme.Codecs.DecoderToVersion(scheme.Codecs.UniversalDeserializer(), gvs)
}

// ApplyResource creates or updates given resource using server-side-apply.
func ApplyResource(dynclient dynamic.Interface, mapper meta.RESTMapper, data []byte, rev int64, forceConflicts bool) error {
	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return err
	}
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[AnnotationResourceRevision] = strconv.FormatInt(rev, 10)
	obj.SetAnnotations(ann)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dynclient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dynclient.Resource(mapping.Resource)
	}
	buf := &bytes.Buffer{}
	if err := unstructured.UnstructuredJSONScheme.Encode(obj, buf); err != nil {
		return err
	}
	if log.Enabled(log.LvDebug) {
		log.Debug("resource-apply", map[string]interface{}{
			"gvk":       gvk.String(),
			"gvr":       mapping.Resource.String(),
			"namespace": obj.GetNamespace(),
			"name":      obj.GetName(),
			"data":      string(buf.Bytes()),
		})
	}

	_, err = dr.Patch(obj.GetName(), types.ApplyPatchType, buf.Bytes(), metav1.PatchOptions{
		FieldManager: "cke",
		Force:        &forceConflicts,
	})
	return err
}

// ParseResource parses YAML string.
func ParseResource(data []byte) (key string, err error) {
	obj, gvk, err := resourceDecoder.Decode(data, nil, nil)
	if err != nil {
		return "", err
	}

	switch o := obj.(type) {
	case *corev1.Namespace:
		return o.Kind + "/" + o.Name, nil
	case *corev1.ServiceAccount:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *corev1.ConfigMap:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *corev1.Service:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *policyv1beta1.PodSecurityPolicy:
		return o.Kind + "/" + o.Name, nil
	case *networkingv1.NetworkPolicy:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *rbacv1.Role:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *rbacv1.RoleBinding:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *rbacv1.ClusterRole:
		return o.Kind + "/" + o.Name, nil
	case *rbacv1.ClusterRoleBinding:
		return o.Kind + "/" + o.Name, nil
	case *appsv1.Deployment:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *appsv1.DaemonSet:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *batchv1beta1.CronJob:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	case *policyv1beta1.PodDisruptionBudget:
		return o.Kind + "/" + o.Namespace + "/" + o.Name, nil
	}

	return "", fmt.Errorf("unsupported type: %s", gvk.String())
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
func (d ResourceDefinition) NeedUpdate(rs *ResourceStatus) bool {
	if rs == nil {
		return true
	}
	curRev, ok := rs.Annotations[AnnotationResourceRevision]
	if !ok {
		return true
	}
	if curRev != strconv.FormatInt(d.Revision, 10) {
		return true
	}

	if d.Image == "" {
		return false
	}

	curImage, ok := rs.Annotations[AnnotationResourceImage]
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
