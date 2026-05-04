package vm

import "strings"

// InsertURL 拼接 vmauth 多租户写入路径前缀（与 dev-plan 示例一致：/insert/{vmuser_id}）。
func InsertURL(baseURL, vmuserID string) string {
	if baseURL == "" || vmuserID == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/insert/" + vmuserID
}

// SelectURL 拼接 vmauth 多租户查询路径前缀。Grafana / 后端查询 API 应使用
// 该租户级入口，而不是绕过 vmauth 直接访问全局 vmselect。
func SelectURL(baseURL, vmuserID string) string {
	if baseURL == "" || vmuserID == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/select/" + vmuserID + "/prometheus"
}

func APIURL(selectURL, path string) string {
	if selectURL == "" {
		return ""
	}
	return strings.TrimRight(selectURL, "/") + "/" + strings.TrimLeft(path, "/")
}
