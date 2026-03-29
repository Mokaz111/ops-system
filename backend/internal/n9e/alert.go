package n9e

import (
	"context"
	"encoding/json"
	"fmt"
)

func toN9ESeverity(level string) int {
	switch level {
	case "critical":
		return 1
	case "warning":
		return 2
	default:
		return 3
	}
}

// CreateAlertRule 创建告警规则（N9E v8 要求数组），返回 N9E 侧 ID。
func (c *Client) CreateAlertRule(ctx context.Context, teamID int64, payload map[string]any) (int64, error) {
	if !c.Enabled() || len(payload) == 0 {
		return 0, nil
	}

	rule := c.buildN9ERule(teamID, payload)

	path := fmt.Sprintf("%s/busi-group/%d/alert-rules", c.prefix, teamID)
	dat, err := c.RequestJSON(ctx, "POST", path, []map[string]any{rule})
	if err != nil {
		return 0, err
	}

	var result map[string]any
	if err := json.Unmarshal(dat, &result); err == nil {
		for _, v := range result {
			if s, ok := v.(string); ok && s == "" {
				continue
			}
		}
	}

	rules, err := c.listRulesInGroup(ctx, teamID)
	if err != nil {
		return 0, nil
	}
	name, _ := payload["name"].(string)
	if name == "" {
		name, _ = payload["rule_name"].(string)
	}
	for _, r := range rules {
		if rName, ok := r["name"].(string); ok && rName == name {
			if id, ok := r["id"].(float64); ok {
				return int64(id), nil
			}
		}
	}
	return 0, nil
}

func (c *Client) listRulesInGroup(ctx context.Context, groupID int64) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/busi-group/%d/alert-rules", c.prefix, groupID)
	dat, err := c.RequestJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var rules []map[string]any
	if err := json.Unmarshal(dat, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Client) buildN9ERule(teamID int64, payload map[string]any) map[string]any {
	name, _ := payload["name"].(string)
	if name == "" {
		name, _ = payload["rule_name"].(string)
	}
	query, _ := payload["query"].(string)
	if q, ok := payload["prom_ql"].(string); ok && q != "" {
		query = q
	}
	annotations, _ := payload["annotations"].(string)
	level, _ := payload["level"].(string)
	severity := toN9ESeverity(level)
	if s, ok := payload["severity"].(int); ok {
		severity = s
	}

	disabled := 0
	if enabled, ok := payload["enabled"].(bool); ok && !enabled {
		disabled = 1
	}

	rule := map[string]any{
		"name":               name,
		"note":               annotations,
		"severity":           severity,
		"disabled":           disabled,
		"prom_eval_interval": 15,
		"prom_for_duration":  60,
		"notify_repeat_step": 60,
		"cate":               "prometheus",
		"datasource_ids":     []int{0},
		"notify_channels":    []string{},
		"notify_groups":      []string{},
		"callbacks":          []string{},
		"append_tags":        []string{},
		"annotations":        map[string]string{},
		"extra_config":       map[string]any{},
		"rule_config": map[string]any{
			"queries": []map[string]any{
				{
					"prom_ql":  query,
					"severity": severity,
				},
			},
		},
	}
	return rule
}

// UpdateAlertRule 更新 N9E 侧告警规则。
func (c *Client) UpdateAlertRule(ctx context.Context, ruleID int64, payload map[string]any) error {
	if !c.Enabled() || ruleID <= 0 {
		return nil
	}

	fields := map[string]any{}
	if name, ok := payload["name"].(string); ok && name != "" {
		fields["name"] = name
	}
	if name, ok := payload["rule_name"].(string); ok && name != "" {
		fields["name"] = name
	}
	if q, ok := payload["query"].(string); ok && q != "" {
		fields["rule_config"] = map[string]any{
			"queries": []map[string]any{
				{"prom_ql": q},
			},
		}
	}
	if level, ok := payload["level"].(string); ok && level != "" {
		fields["severity"] = toN9ESeverity(level)
	}
	if ann, ok := payload["annotations"].(string); ok {
		fields["note"] = ann
	}
	if enabled, ok := payload["enabled"].(bool); ok {
		if enabled {
			fields["disabled"] = 0
		} else {
			fields["disabled"] = 1
		}
	}

	path := fmt.Sprintf("%s/alert-rules", c.prefix)
	body := map[string]any{
		"ids":    []int64{ruleID},
		"fields": fields,
	}
	return c.doJSON(ctx, "PUT", path, body, nil)
}

// DeleteAlertRule 删除 N9E 侧告警规则（N9E v8 使用 idsForm）。
func (c *Client) DeleteAlertRule(ctx context.Context, ruleID int64) error {
	if !c.Enabled() || ruleID <= 0 {
		return nil
	}
	path := fmt.Sprintf("%s/alert-rules", c.prefix)
	body := map[string]any{"ids": []int64{ruleID}}
	return c.doJSON(ctx, "DELETE", path, body, nil)
}

// GetActiveAlerts 获取当前活跃告警事件（N9E v8 路径 /alert-cur-events/list）。
func (c *Client) GetActiveAlerts(ctx context.Context, limit int) (json.RawMessage, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	path := fmt.Sprintf("%s/alert-cur-events/list?p=1&limit=%d", c.prefix, limit)
	return c.RequestJSON(ctx, "GET", path, nil)
}

// GetHistoryAlerts 获取历史告警事件。
func (c *Client) GetHistoryAlerts(ctx context.Context, limit int) (json.RawMessage, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	path := fmt.Sprintf("%s/alert-his-events/list?p=1&limit=%d", c.prefix, limit)
	return c.RequestJSON(ctx, "GET", path, nil)
}
