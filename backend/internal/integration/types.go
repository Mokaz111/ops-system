package integration

// TemplateSpec 定义一个模版版本在 DB 中 JSONB 字段的完整结构。
// 以 Go 结构体统一约束，所有字段对应:
//   - CollectorSpec → TemplateSpec.Collector
//   - AlertSpec     → TemplateSpec.Alert
//   - DashboardSpec → TemplateSpec.Dashboards
//   - Variables     → TemplateSpec.Variables
type TemplateSpec struct {
	Variables  []Variable        `json:"variables,omitempty"`
	Collector  CollectorSpec     `json:"collector"`
	Alert      AlertSpec         `json:"alert"`
	Dashboards []DashboardSpec   `json:"dashboards"`
}

// Variable 模版变量定义。
type Variable struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"`     // string / int / bool / enum
	Default  string `json:"default"`
	Required bool   `json:"required"`
	Help     string `json:"help,omitempty"`
	Options  []string `json:"options,omitempty"` // type=enum 时可选项
}

// CollectorSpec 采集部分；每个 resource 是一个 K8s CR / ConfigMap 的 Go 模版片段。
type CollectorSpec struct {
	Resources []ResourceTemplate `json:"resources"`
}

// AlertSpec 告警部分。VMRule 以模版形式表达；N9E 规则为占位。
type AlertSpec struct {
	Targets []string           `json:"targets"` // ["vmrule"] / ["vmrule","n9e"]
	VMRules []ResourceTemplate `json:"vmrules"`
	N9E     []N9ERule          `json:"n9e,omitempty"`
}

// N9ERule N9E 规则占位。
type N9ERule struct {
	Name string `json:"name"`
	Expr string `json:"expr"`
	For  string `json:"for"`
}

// DashboardSpec Grafana dashboard JSON + 元数据。
type DashboardSpec struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
	JSON  string `json:"json"` // 完整 dashboard JSON，含 `{{ .Values.xxx }}` 占位
}

// ResourceTemplate 一个可渲染的 K8s 资源模版。
type ResourceTemplate struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	Manifest   string `json:"manifest"` // YAML 文本，使用 Go text/template 语法
}

// RenderedResource 渲染结果。
type RenderedResource struct {
	Part       string `json:"part"`        // collector / vmrule / dashboard / n9e
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	YAML       string `json:"yaml"`
	Dashboard  string `json:"dashboard,omitempty"` // dashboard JSON
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
	Spec      TemplateSpec
	Values    map[string]string
	Ctx       RenderContext
	Parts     []string // 仅渲染这些部件；为空表示全部
}
