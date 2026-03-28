package n9e

import (
	"context"
)

// N9EUser 创建夜莺用户请求（§2.4.3，按需扩展字段）。
type N9EUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// CreateUser 创建用户（团队绑定等依实际 API 再补）。
func (c *Client) CreateUser(ctx context.Context, u *N9EUser) error {
	if !c.Enabled() || u == nil {
		return nil
	}
	return c.doJSON(ctx, "POST", c.prefix+"/users", u, nil)
}
