package integration

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ExtractedMetric 解析出的指标。
type ExtractedMetric struct {
	Name               string
	AppearsInCollector bool
	AppearsInDashboard bool
	AppearsInAlert     bool
	Panels             []PanelRef
}

// PanelRef Dashboard 面板引用。
type PanelRef struct {
	DashboardUID string `json:"dashboard_uid"`
	PanelID      int    `json:"panel_id"`
	Title        string `json:"title"`
	Expr         string `json:"expr"`
}

// metricNameRE 匹配 PromQL/MetricsQL 中的指标名。
// 规则：字母/下划线开头，后跟 [A-Za-z0-9_:] ；排除 by/group_left 等关键字由调用方处理。
var metricNameRE = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_:]*`)

// reservedPromKeywords PromQL 关键字 / 函数；解析时排除。
var reservedPromKeywords = map[string]struct{}{
	"by": {}, "without": {}, "on": {}, "ignoring": {}, "group_left": {}, "group_right": {},
	"and": {}, "or": {}, "unless": {}, "offset": {},
	"sum": {}, "avg": {}, "min": {}, "max": {}, "count": {}, "stddev": {}, "stdvar": {}, "topk": {}, "bottomk": {}, "quantile": {},
	"rate": {}, "irate": {}, "increase": {}, "delta": {}, "deriv": {}, "predict_linear": {},
	"histogram_quantile": {}, "histogram_sum": {}, "histogram_count": {}, "histogram_avg": {}, "histogram_stddev": {},
	"label_replace": {}, "label_join": {}, "vector": {}, "scalar": {}, "time": {}, "timestamp": {},
	"absent": {}, "absent_over_time": {}, "changes": {}, "resets": {}, "round": {}, "clamp": {}, "clamp_min": {}, "clamp_max": {},
	"avg_over_time": {}, "sum_over_time": {}, "min_over_time": {}, "max_over_time": {}, "count_over_time": {}, "quantile_over_time": {},
	"le": {}, "ge": {}, "lt": {}, "gt": {}, "eq": {}, "ne": {},
	"inf": {}, "nan": {}, "true": {}, "false": {},
	"if": {}, "ifnot": {}, "else": {}, "for": {},
}

// ExtractFromExpr 从单个表达式抽取候选指标名。
func ExtractFromExpr(expr string) []string {
	if strings.TrimSpace(expr) == "" {
		return nil
	}
	tokens := metricNameRE.FindAllString(expr, -1)
	seen := map[string]bool{}
	var out []string
	for _, t := range tokens {
		if _, ok := reservedPromKeywords[t]; ok {
			continue
		}
		if isNumericLike(t) {
			continue
		}
		// 排除全大写字符串（可能是常量）、长度 < 3 的误匹配。
		if len(t) < 3 {
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func isNumericLike(s string) bool {
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

// ExtractFromSpec 扫描整个 TemplateSpec，返回指标候选集合。
func ExtractFromSpec(spec TemplateSpec) map[string]*ExtractedMetric {
	result := map[string]*ExtractedMetric{}

	ensure := func(name string) *ExtractedMetric {
		if m, ok := result[name]; ok {
			return m
		}
		m := &ExtractedMetric{Name: name}
		result[name] = m
		return m
	}

	// Collector：解析 manifest 中 `__name__` 标签或 scrape_configs relabeling 中指向的指标名；
	// 简化：扫描整个 manifest 中形如 metric_name 的 token，只取在注释/metric 列表中的。
	// 更稳妥做法：对 VMPodScrape 的 metricRelabelConfigs.sourceLabels=__name__ 的 regex 做提取。
	// 这里保守：对 manifest 中 # metrics: 注释行后续单词做提取。
	for _, r := range spec.Collector.Resources {
		for _, name := range extractMetricsCommentBlock(r.Manifest) {
			ensure(name).AppearsInCollector = true
		}
	}

	// VMRule：解析 groups[].rules[].expr
	for _, r := range spec.Alert.VMRules {
		for _, expr := range extractExprsFromVMRuleYAML(r.Manifest) {
			for _, name := range ExtractFromExpr(expr) {
				ensure(name).AppearsInAlert = true
			}
		}
	}

	// Dashboard：解析 targets[].expr
	for _, d := range spec.Dashboards {
		panels := extractPanelsFromDashboardJSON(d.UID, d.JSON)
		for _, p := range panels {
			for _, name := range ExtractFromExpr(p.Expr) {
				m := ensure(name)
				m.AppearsInDashboard = true
				m.Panels = append(m.Panels, p)
			}
		}
	}

	return result
}

// metricsCommentRE 识别形如 `# metrics: node_cpu_seconds_total, node_load1` 的注释块。
var metricsCommentRE = regexp.MustCompile(`(?i)#\s*metrics?\s*:\s*([A-Za-z0-9_,\s:]+)`)

func extractMetricsCommentBlock(body string) []string {
	var out []string
	for _, m := range metricsCommentRE.FindAllStringSubmatch(body, -1) {
		if len(m) < 2 {
			continue
		}
		for _, name := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			name = strings.TrimSpace(name)
			if name != "" {
				out = append(out, name)
			}
		}
	}
	return out
}

// extractExprsFromVMRuleYAML 从 VMRule manifest 里抓 `expr:` 行。
// 简单文本解析，避免拉 YAML 依赖。
func extractExprsFromVMRuleYAML(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "expr:") {
			v := strings.TrimSpace(strings.TrimPrefix(trim, "expr:"))
			v = strings.Trim(v, "\"'`|>")
			v = strings.TrimSpace(v)
			if v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// extractPanelsFromDashboardJSON 解析 Grafana dashboard JSON 的 panels[].targets[].expr。
func extractPanelsFromDashboardJSON(uid, body string) []PanelRef {
	var out []PanelRef
	var dash struct {
		UID    string `json:"uid"`
		Title  string `json:"title"`
		Panels []struct {
			ID      int    `json:"id"`
			Title   string `json:"title"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	}
	if err := json.Unmarshal([]byte(body), &dash); err != nil {
		return out
	}
	dashUID := uid
	if dash.UID != "" {
		dashUID = dash.UID
	}
	for _, p := range dash.Panels {
		for _, t := range p.Targets {
			if strings.TrimSpace(t.Expr) == "" {
				continue
			}
			out = append(out, PanelRef{
				DashboardUID: dashUID,
				PanelID:      p.ID,
				Title:        p.Title,
				Expr:         t.Expr,
			})
		}
	}
	return out
}
