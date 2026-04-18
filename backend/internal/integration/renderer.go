package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// Renderer 模版渲染器。纯函数语义，可复用。
type Renderer struct{}

// NewRenderer 构造。
func NewRenderer() *Renderer { return &Renderer{} }

// ParseSpec 从 DB 三段 JSONB（collector/alert/dashboard）+ variables 字符串还原 TemplateSpec。
// 允许任意段为空字符串（→ 零值）。
func ParseSpec(collector, alert, dashboard, variables string) (TemplateSpec, error) {
	var spec TemplateSpec
	if s := strings.TrimSpace(collector); s != "" {
		if err := json.Unmarshal([]byte(s), &spec.Collector); err != nil {
			return spec, fmt.Errorf("parse collector: %w", err)
		}
	}
	if s := strings.TrimSpace(alert); s != "" {
		if err := json.Unmarshal([]byte(s), &spec.Alert); err != nil {
			return spec, fmt.Errorf("parse alert: %w", err)
		}
	}
	if s := strings.TrimSpace(dashboard); s != "" {
		if err := json.Unmarshal([]byte(s), &spec.Dashboards); err != nil {
			return spec, fmt.Errorf("parse dashboards: %w", err)
		}
	}
	if s := strings.TrimSpace(variables); s != "" {
		var v struct {
			Variables []Variable `json:"variables"`
		}
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			// 兼容直接就是数组
			var arr []Variable
			if err2 := json.Unmarshal([]byte(s), &arr); err2 == nil {
				spec.Variables = arr
			} else {
				return spec, fmt.Errorf("parse variables: %w", err)
			}
		} else {
			spec.Variables = v.Variables
		}
	}
	return spec, nil
}

// Render 按需渲染模版为资源列表。
func (r *Renderer) Render(in RenderInput) ([]RenderedResource, error) {
	wants := toSet(in.Parts)
	merged := mergeValues(in.Spec.Variables, in.Values)
	root := map[string]any{
		"Values":  merged,
		"Context": in.Ctx,
	}
	var out []RenderedResource

	if wants["collector"] || len(wants) == 0 {
		for _, rt := range in.Spec.Collector.Resources {
			rendered, err := execTpl(rt.Name, rt.Manifest, root)
			if err != nil {
				return nil, fmt.Errorf("render collector %s: %w", rt.Name, err)
			}
			out = append(out, RenderedResource{
				Part:       "collector",
				Kind:       rt.Kind,
				APIVersion: rt.APIVersion,
				Name:       rt.Name,
				YAML:       rendered,
			})
		}
	}

	if wants["vmrule"] || wants["alert"] || len(wants) == 0 {
		if containsStr(in.Spec.Alert.Targets, "vmrule") || len(in.Spec.Alert.Targets) == 0 {
			for _, rt := range in.Spec.Alert.VMRules {
				rendered, err := execTpl(rt.Name, rt.Manifest, root)
				if err != nil {
					return nil, fmt.Errorf("render vmrule %s: %w", rt.Name, err)
				}
				out = append(out, RenderedResource{
					Part:       "vmrule",
					Kind:       rt.Kind,
					APIVersion: rt.APIVersion,
					Name:       rt.Name,
					YAML:       rendered,
				})
			}
		}
	}

	if wants["dashboard"] || len(wants) == 0 {
		for _, d := range in.Spec.Dashboards {
			rendered, err := execTpl("dashboard-"+d.UID, d.JSON, root)
			if err != nil {
				return nil, fmt.Errorf("render dashboard %s: %w", d.UID, err)
			}
			out = append(out, RenderedResource{
				Part:      "dashboard",
				Kind:      "GrafanaDashboard",
				Name:      d.UID,
				Dashboard: rendered,
			})
		}
	}

	return out, nil
}

func execTpl(name, body string, root any) (string, error) {
	t, err := template.New(name).Option("missingkey=zero").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, root); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func mergeValues(defs []Variable, given map[string]string) map[string]string {
	out := map[string]string{}
	for _, v := range defs {
		if v.Default != "" {
			out[v.Name] = v.Default
		}
	}
	for k, v := range given {
		out[k] = v
	}
	return out
}

func toSet(arr []string) map[string]bool {
	s := map[string]bool{}
	for _, x := range arr {
		if x != "" {
			s[x] = true
		}
	}
	return s
}

func containsStr(arr []string, x string) bool {
	for _, v := range arr {
		if v == x {
			return true
		}
	}
	return false
}
