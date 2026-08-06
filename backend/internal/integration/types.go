package integration

// TemplateSpec 定义一个模版版本在 DB 中 JSONB 字段的完整结构。
type TemplateSpec struct {
	Variables  []Variable      `json:"variables,omitempty"`
	Collector  CollectorSpec   `json:"collector"`
	Alert      AlertSpec       `json:"alert"`
	Dashboards []DashboardSpec `json:"dashboards"`
}

// Variable 模版变量定义。
type Variable struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // string / int / bool / enum
	Default  string   `json:"default"`
	Required bool     `json:"required"`
	Help     string   `json:"help,omitempty"`
	Options  []string `json:"options,omitempty"`
}

// CollectorSpec 采集部分；每个 resource 是一个 K8s CR / ConfigMap 的 Go 模版片段。
type CollectorSpec struct {
	Resources []ResourceTemplate `json:"resources"`
	Workloads []ResourceTemplate `json:"workloads,omitempty"`
}

// AlertSpec 告警部分。VMRule 以模版形式表达。
type AlertSpec struct {
	Targets []string           `json:"targets"` // ["vmrule"]
	VMRules []ResourceTemplate `json:"vmrules"`
}

// DashboardSpec Grafana dashboard JSON + 元数据。
type DashboardSpec struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
	JSON  string `json:"json"`
}

// ResourceTemplate 一个可渲染的 K8s 资源模版。
type ResourceTemplate struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	Manifest   string `json:"manifest"`
}

// RenderedResource 渲染结果。
type RenderedResource struct {
	Part       string `json:"part"` // collector / vmrule / dashboard / workload
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	YAML       string `json:"yaml"`
	Dashboard  string `json:"dashboard,omitempty"`
}

// RenderContext 渲染上下文（平台补齐的内置变量）。
type RenderContext struct {
	TenantID     string `json:"tenant_id"`
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	Namespace    string `json:"namespace"`
	VMAgentURL   string `json:"vmagent_url"`
	GrafanaOrgID int64  `json:"grafana_org_id"`
}

// RenderInput 渲染一次的完整输入。
type RenderInput struct {
	Spec   TemplateSpec
	Values map[string]string
	Ctx    RenderContext
	Parts  []string
}
