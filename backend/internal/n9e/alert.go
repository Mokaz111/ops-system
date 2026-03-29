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

// CreateAlertRule 创建告警规则，返回 N9E 侧 ID。
func (c *Client) CreateAlertRule(ctx context.Context, teamID int64, payload map[string]any) (int64, error) {
	if !c.Enabled() || len(payload) == 0 {
		return 0, nil
	}
	if teamID > 0 {
		payload["group_id"] = teamID
	}
	if level, ok := payload["level"].(string); ok {
		payload["severity"] = toN9ESeverity(level)
		delete(payload, "level")
	}
	if q, ok := payload["query"].(string); ok {
		payload["prom_ql"] = q
		delete(payload, "query")
	}
	if enabled, ok := payload["enabled"].(bool); ok {
		if enabled {
			payload["disabled"] = 0
		} else {
			payload["disabled"] = 1
		}
		delete(payload, "enabled")
	}

	dat, err := c.RequestJSON(ctx, "POST", c.prefix+"/busi-groups/"+fmt.Sprintf("%d", teamID)+"/alert-rules", payload)
	if err != nil {
		return 0, err
	}
	id, err := parseDatAsInt64(dat)
	if err != nil {
		var m map[string]any
		if err2 := json.Unmarshal(dat, &m); err2 == nil {
			if v, ok := m["id"].(float64); ok {
				return int64(v), nil
			}
		}
		return 0, nil
	}
	return id, nil
}

// UpdateAlertRule 更新 N9E 侧告警规则。
func (c *Client) UpdateAlertRule(ctx context.Context, ruleID int64, payload map[string]any) error {
	if !c.Enabled() || ruleID <= 0 {
		return nil
	}
	if level, ok := payload["level"].(string); ok {
		payload["severity"] = toN9ESeverity(level)
		delete(payload, "level")
	}
	if q, ok := payload["query"].(string); ok {
		payload["prom_ql"] = q
		delete(payload, "query")
	}
	if enabled, ok := payload["enabled"].(bool); ok {
		if enabled {
			payload["disabled"] = 0
		} else {
			payload["disabled"] = 1
		}
		delete(payload, "enabled")
	}
	path := fmt.Sprintf("%s/alert-rules/%d", c.prefix, ruleID)
	return c.doJSON(ctx, "PUT", path, payload, nil)
}

// DeleteAlertRule 删除 N9E 侧告警规则。
func (c *Client) DeleteAlertRule(ctx context.Context, ruleID int64) error {
	if !c.Enabled() || ruleID <= 0 {
		return nil
	}
	path := fmt.Sprintf("%s/alert-rules/%d", c.prefix, ruleID)
	_, err := c.RequestJSON(ctx, "DELETE", path, nil)
	return err
}

// GetActiveAlerts 获取当前活跃告警事件。
func (c *Client) GetActiveAlerts(ctx context.Context, limit int) (json.RawMessage, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	path := fmt.Sprintf("%s/alert-cur-events?limit=%d", c.prefix, limit)
	return c.RequestJSON(ctx, "GET", path, nil)
}
