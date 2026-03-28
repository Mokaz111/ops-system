package vm

import "strings"

// InsertURL 拼接 vmauth 多租户写入路径前缀（与 dev-plan 示例一致：/insert/{vmuser_id}）。
func InsertURL(baseURL, vmuserID string) string {
	if baseURL == "" || vmuserID == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/insert/" + vmuserID
}
