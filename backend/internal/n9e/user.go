package n9e

import (
	"context"
)

// N9EUser 创建夜莺用户请求（N9E v8 格式）。
type N9EUser struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Nickname string   `json:"nickname,omitempty"`
	Email    string   `json:"email,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

// CreateUser 创建 N9E 用户。
func (c *Client) CreateUser(ctx context.Context, u *N9EUser) error {
	if !c.Enabled() || u == nil {
		return nil
	}
	if len(u.Roles) == 0 {
		u.Roles = []string{"Standard"}
	}
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	return c.doJSON(ctx, "POST", c.prefix+"/users", u, nil)
}
