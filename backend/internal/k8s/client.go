package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memcached "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	sigyaml "sigs.k8s.io/yaml"
)

// Client 租户命名空间与配额、网络策略 + 通用 CR 管理。
type Client struct {
	cs     kubernetes.Interface
	dyn    dynamic.Interface
	disco  discovery.CachedDiscoveryInterface
	mapper meta.RESTMapper
}

// NewClient 构建 client-go；inCluster 为 true 时使用 Pod 内 ServiceAccount。
func NewClient(kubeconfig string, inCluster bool) (*Client, error) {
	var cfg *rest.Config
	var err error
	if inCluster {
		cfg, err = rest.InClusterConfig()
	} else {
		if strings.TrimSpace(kubeconfig) == "" {
			return nil, fmt.Errorf("kubeconfig path is empty and incluster is false")
		}
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	discoClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cached := memcached.NewMemCacheClient(discoClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cached)
	return &Client{cs: cs, dyn: dc, disco: cached, mapper: mapper}, nil
}

// EnsureNamespace 创建命名空间（已存在则忽略）。
func (c *Client) EnsureNamespace(ctx context.Context, name string, labels map[string]string) error {
	if labels == nil {
		labels = map[string]string{}
	}
	_, err := c.cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err = c.cs.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

// DeleteNamespace 删除命名空间。
func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.cs.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "not found") {
		return nil
	}
	return err
}

// ApplyResourceQuota 根据 quota_config JSON 应用 ResourceQuota；空则使用轻量默认。
func (c *Client) ApplyResourceQuota(ctx context.Context, namespace, quotaJSON string) error {
	hard, err := parseQuotaHard(quotaJSON)
	if err != nil {
		return err
	}
	name := "ops-tenant-quota"
	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{Hard: hard},
	}
	_, err = c.cs.CoreV1().ResourceQuotas(namespace).Create(ctx, rq, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return err
	}
	cur, gerr := c.cs.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if gerr != nil {
		return gerr
	}
	cur.Spec.Hard = hard
	_, err = c.cs.CoreV1().ResourceQuotas(namespace).Update(ctx, cur, metav1.UpdateOptions{})
	return err
}

func parseQuotaHard(quotaJSON string) (corev1.ResourceList, error) {
	quotaJSON = strings.TrimSpace(quotaJSON)
	if quotaJSON == "" {
		return corev1.ResourceList{
			corev1.ResourceRequestsCPU:    resource.MustParse("500m"),
			corev1.ResourceRequestsMemory: resource.MustParse("512Mi"),
			corev1.ResourceLimitsCPU:      resource.MustParse("2"),
			corev1.ResourceLimitsMemory:   resource.MustParse("2Gi"),
		}, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(quotaJSON), &raw); err != nil {
		return nil, fmt.Errorf("quota_config json: %w", err)
	}
	hard := corev1.ResourceList{}
	// 支持扁平键：requests_cpu / limits_memory 或嵌套 requests: {cpu: ...}
	if v, ok := stringVal(raw, "requests_cpu"); ok {
		hard[corev1.ResourceRequestsCPU] = resource.MustParse(v)
	}
	if v, ok := stringVal(raw, "requests_memory"); ok {
		hard[corev1.ResourceRequestsMemory] = resource.MustParse(v)
	}
	if v, ok := stringVal(raw, "limits_cpu"); ok {
		hard[corev1.ResourceLimitsCPU] = resource.MustParse(v)
	}
	if v, ok := stringVal(raw, "limits_memory"); ok {
		hard[corev1.ResourceLimitsMemory] = resource.MustParse(v)
	}
	if req, ok := raw["requests"].(map[string]interface{}); ok {
		if v, ok := stringVal(req, "cpu"); ok {
			hard[corev1.ResourceRequestsCPU] = resource.MustParse(v)
		}
		if v, ok := stringVal(req, "memory"); ok {
			hard[corev1.ResourceRequestsMemory] = resource.MustParse(v)
		}
	}
	if lim, ok := raw["limits"].(map[string]interface{}); ok {
		if v, ok := stringVal(lim, "cpu"); ok {
			hard[corev1.ResourceLimitsCPU] = resource.MustParse(v)
		}
		if v, ok := stringVal(lim, "memory"); ok {
			hard[corev1.ResourceLimitsMemory] = resource.MustParse(v)
		}
	}
	if len(hard) == 0 {
		return parseQuotaHard("")
	}
	return hard, nil
}

func stringVal(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok && s != ""
}

// ScaleDeployment 修改 Deployment 副本数。
func (c *Client) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	deploy, err := c.cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deploy.Spec.Replicas = &replicas
	_, err = c.cs.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

// PatchCustomResourceSpec 以 MergePatch 方式更新 CR 的 spec 字段。
func (c *Client) PatchCustomResourceSpec(
	ctx context.Context,
	group, version, resource, namespace, name string,
	spec map[string]interface{},
) error {
	if c.dyn == nil {
		return fmt.Errorf("dynamic kubernetes client is not configured")
	}
	patchBody := map[string]interface{}{"spec": spec}
	raw, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	_, err = c.dyn.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, raw, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

// GetCustomResource 返回任意 CR 的内容。
func (c *Client) GetCustomResource(
	ctx context.Context,
	group, version, resource, namespace, name string,
) (*unstructured.Unstructured, error) {
	if c.dyn == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not configured")
	}
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	obj, err := c.dyn.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return obj, nil
}

// ResizePVC 修改 PVC 存储大小。
func (c *Client) ResizePVC(ctx context.Context, namespace, name, newSize string) error {
	pvc, err := c.cs.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(newSize)
	_, err = c.cs.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	return err
}

// ResolveGVR 通过 RESTMapper 将 GVK 映射成 GVR。
func (c *Client) ResolveGVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, bool, error) {
	if c.mapper == nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("rest mapper is not configured")
	}
	m, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}
	namespaced := m.Scope != nil && m.Scope.Name() == meta.RESTScopeNameNamespace
	return m.Resource, namespaced, nil
}

// ResolveGVRByString 用字符串形式的 apiVersion / kind 解析 GVR。
func (c *Client) ResolveGVRByString(apiVersion, kind string) (schema.GroupVersionResource, bool, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("parse apiVersion %s: %w", apiVersion, err)
	}
	return c.ResolveGVR(gv.WithKind(kind))
}

// InvalidateMapperCache 强制刷新 discovery 缓存（CRD 新增时使用）。
func (c *Client) InvalidateMapperCache() {
	if c.disco != nil {
		c.disco.Invalidate()
	}
}

// ApplyYAMLResult 单个资源 apply 结果。
type ApplyYAMLResult struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	Action     string // created | updated
	UID        string
}

// ApplyYAML 将 YAML 文本（可能包含多个 --- 分隔文档）依次 apply 到集群。
// defaultNamespace 在 YAML 未指定 metadata.namespace 时生效（对 namespaced 资源）。
func (c *Client) ApplyYAML(ctx context.Context, yamlText, defaultNamespace string) ([]ApplyYAMLResult, error) {
	docs := splitYAMLDocs(yamlText)
	results := make([]ApplyYAMLResult, 0, len(docs))
	for _, doc := range docs {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" {
			continue
		}
		obj := &unstructured.Unstructured{}
		jsonBytes, err := sigyaml.YAMLToJSON([]byte(trimmed))
		if err != nil {
			return results, fmt.Errorf("yaml to json: %w", err)
		}
		if err := obj.UnmarshalJSON(jsonBytes); err != nil {
			return results, fmt.Errorf("unmarshal object: %w", err)
		}
		if obj.GetKind() == "" {
			continue
		}
		res, err := c.applyUnstructured(ctx, obj, defaultNamespace)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

// applyUnstructured 对单个 unstructured 对象执行 create-or-update。
func (c *Client) applyUnstructured(ctx context.Context, obj *unstructured.Unstructured, defaultNamespace string) (ApplyYAMLResult, error) {
	res := ApplyYAMLResult{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
	}
	if c.dyn == nil || c.mapper == nil {
		return res, fmt.Errorf("dynamic client / mapper is not configured")
	}
	gvk := obj.GroupVersionKind()
	gvr, namespaced, err := c.ResolveGVR(gvk)
	if err != nil {
		return res, fmt.Errorf("resolve gvr %s: %w", gvk.String(), err)
	}
	ns := obj.GetNamespace()
	if namespaced && strings.TrimSpace(ns) == "" {
		ns = defaultNamespace
		obj.SetNamespace(ns)
		res.Namespace = ns
	}
	var ri dynamic.ResourceInterface
	if namespaced {
		ri = c.dyn.Resource(gvr).Namespace(ns)
	} else {
		ri = c.dyn.Resource(gvr)
	}
	existing, err := ri.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return res, fmt.Errorf("get %s/%s: %w", gvk.Kind, obj.GetName(), err)
		}
		created, err := ri.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return res, fmt.Errorf("create %s/%s: %w", gvk.Kind, obj.GetName(), err)
		}
		res.Action = "created"
		res.UID = string(created.GetUID())
		return res, nil
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	updated, err := ri.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return res, fmt.Errorf("update %s/%s: %w", gvk.Kind, obj.GetName(), err)
	}
	res.Action = "updated"
	res.UID = string(updated.GetUID())
	return res, nil
}

// DeleteByGVK 通过 RESTMapper 解析 GVR 后删除资源，NotFound 视为成功。
func (c *Client) DeleteByGVK(ctx context.Context, apiVersion, kind, namespace, name string) error {
	if c.dyn == nil || c.mapper == nil {
		return fmt.Errorf("dynamic client / mapper is not configured")
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return fmt.Errorf("parse apiVersion %s: %w", apiVersion, err)
	}
	gvr, namespaced, err := c.ResolveGVR(gv.WithKind(kind))
	if err != nil {
		return fmt.Errorf("resolve gvr: %w", err)
	}
	var ri dynamic.ResourceInterface
	if namespaced {
		ri = c.dyn.Resource(gvr).Namespace(namespace)
	} else {
		ri = c.dyn.Resource(gvr)
	}
	if err := ri.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// splitYAMLDocs 按 --- 切分 YAML 多文档；保留 anchor，不处理转义。
func splitYAMLDocs(yamlText string) []string {
	normalized := strings.ReplaceAll(yamlText, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var docs []string
	var cur strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if cur.Len() > 0 {
				docs = append(docs, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
	}
	if cur.Len() > 0 {
		docs = append(docs, cur.String())
	}
	return docs
}

// ApplyDefaultNetworkPolicy 同命名空间内 Pod 互通，默认拒绝来自其它命名空间入站（可按需收紧）。
func (c *Client) ApplyDefaultNetworkPolicy(ctx context.Context, namespace string) error {
	name := "ops-tenant-default"
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{PodSelector: &metav1.LabelSelector{}},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
	_, err := c.cs.NetworkingV1().NetworkPolicies(namespace).Create(ctx, np, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return nil
	}
	return err
}
