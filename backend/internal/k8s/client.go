package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client 租户命名空间与配额、网络策略。
type Client struct {
	cs kubernetes.Interface
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
	return &Client{cs: cs}, nil
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
