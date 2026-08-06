package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var ErrNoRuleFiles = fmt.Errorf("no yaml rule files found")

// ImportRuleFile 待导入的单个 YAML 规则文件。
type ImportRuleFile struct {
	Name    string
	Content []byte
}

// ImportRuleError 单条规则导入失败的说明。
type ImportRuleError struct {
	File   string `json:"file"`
	Group  string `json:"group"`
	Alert  string `json:"alert"`
	Reason string `json:"reason"`
}

// ImportRulesResult 批量导入结果。
type ImportRulesResult struct {
	Total            int               `json:"total"`
	Created          int               `json:"created"`
	SkippedRecording int               `json:"skipped_recording"`
	SkippedDuplicate int               `json:"skipped_duplicate"`
	Errors           []ImportRuleError `json:"errors"`
}

// Prometheus 规则文件结构（https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/）。
type promRuleFile struct {
	Groups []promRuleGroup `yaml:"groups"`
}

type promRuleGroup struct {
	Name  string     `yaml:"name"`
	Rules []promRule `yaml:"rules"`
}

type promRule struct {
	Alert       string            `yaml:"alert"`
	Record      string            `yaml:"record"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// severityToLevel 把 Prometheus severity label 映射到平台告警级别。
func severityToLevel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "page", "emergency", "error", "fatal":
		return "critical"
	case "info", "none", "notice":
		return "info"
	default:
		return "warning"
	}
}

// ImportRules 批量导入 Prometheus 风格的告警规则。
// recording rules（record: 条目）与目标空间下已存在同名规则会被跳过；
// 单条规则失败不会中断整体导入，失败原因汇总在结果的 errors 中。
func (s *AlertService) ImportRules(ctx context.Context, tenantID uuid.UUID, files []ImportRuleFile) (*ImportRulesResult, error) {
	t, err := s.workspaceRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrWorkspaceNotFound
	}
	if len(files) == 0 {
		return nil, ErrNoRuleFiles
	}

	// 一次性取出该空间已有规则名用于去重。
	existing, _, err := s.ruleRepo.List(ctx, repository.AlertRuleListFilter{
		TenantID: &tenantID,
		Limit:    10000,
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, r := range existing {
		seen[strings.TrimSpace(r.RuleName)] = struct{}{}
	}

	result := &ImportRulesResult{Errors: []ImportRuleError{}}
	for _, f := range files {
		var parsed promRuleFile
		if err := yaml.Unmarshal(f.Content, &parsed); err != nil {
			result.Errors = append(result.Errors, ImportRuleError{File: f.Name, Reason: "yaml 解析失败: " + err.Error()})
			continue
		}
		if len(parsed.Groups) == 0 {
			result.Errors = append(result.Errors, ImportRuleError{File: f.Name, Reason: "缺少 groups 字段（需要 Prometheus rule file 格式）"})
			continue
		}
		for _, g := range parsed.Groups {
			for _, r := range g.Rules {
				if r.Record != "" {
					result.SkippedRecording++
					continue
				}
				result.Total++
				name := strings.TrimSpace(r.Alert)
				if name == "" {
					result.Errors = append(result.Errors, ImportRuleError{File: f.Name, Group: g.Name, Reason: "缺少 alert 名称"})
					continue
				}
				if strings.TrimSpace(r.Expr) == "" {
					result.Errors = append(result.Errors, ImportRuleError{File: f.Name, Group: g.Name, Alert: name, Reason: "缺少 expr"})
					continue
				}
				if _, dup := seen[name]; dup {
					result.SkippedDuplicate++
					continue
				}

				condition := ""
				if strings.TrimSpace(r.For) != "" {
					b, _ := json.Marshal(map[string]string{"for": strings.TrimSpace(r.For)})
					condition = string(b)
				}
				annotations := ""
				if len(r.Annotations) > 0 {
					b, _ := json.Marshal(r.Annotations)
					annotations = string(b)
				}

				_, err := s.CreateRule(ctx, &CreateAlertRuleRequest{
					TenantID:    tenantID,
					RuleName:    name,
					RuleType:    "metrics",
					Query:       r.Expr,
					Condition:   condition,
					Level:       severityToLevel(r.Labels["severity"]),
					Annotations: annotations,
					Enabled:     true,
				})
				if err != nil {
					result.Errors = append(result.Errors, ImportRuleError{File: f.Name, Group: g.Name, Alert: name, Reason: err.Error()})
					continue
				}
				seen[name] = struct{}{}
				result.Created++
			}
		}
	}
	return result, nil
}
