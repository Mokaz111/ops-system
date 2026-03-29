package n9e

import (
	"context"
	"encoding/json"
	"fmt"
)

func parseDatAsInt64(dat json.RawMessage) (int64, error) {
	if len(dat) == 0 || string(dat) == "null" {
		return 0, fmt.Errorf("empty dat")
	}
	var n int64
	if err := json.Unmarshal(dat, &n); err == nil {
		return n, nil
	}
	var o struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(dat, &o); err == nil && o.ID > 0 {
		return o.ID, nil
	}
	return 0, fmt.Errorf("cannot parse id: %s", string(dat))
}

// CreateTeam 创建业务组（N9E v8 busi-group），返回 N9E 侧 ID。
func (c *Client) CreateTeam(ctx context.Context, name, note string) (int64, error) {
	if !c.Enabled() {
		return 0, fmt.Errorf("n9e disabled")
	}
	body := map[string]any{
		"name":         name,
		"label_enable": 0,
		"label_value":  "",
		"members": []map[string]any{
			{"user_group_id": 1, "perm_flag": "rw"},
		},
	}
	dat, err := c.RequestJSON(ctx, "POST", c.prefix+"/busi-groups", body)
	if err != nil {
		return 0, err
	}
	return parseDatAsInt64(dat)
}

// DeleteTeam 删除业务组。
func (c *Client) DeleteTeam(ctx context.Context, teamID int64) error {
	if !c.Enabled() || teamID <= 0 {
		return nil
	}
	path := fmt.Sprintf("%s/busi-group/%d", c.prefix, teamID)
	_, err := c.RequestJSON(ctx, "DELETE", path, nil)
	return err
}
