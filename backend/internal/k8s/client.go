package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client 租户命名空间与配额、网络策略。
type Client struct {
	cs  kubernetes.Interface
	dyn dynamic.Interface
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
	return &Client{cs: cs, dyn: dc}, nil
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
