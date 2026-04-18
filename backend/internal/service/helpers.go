package service

import "encoding/json"

// marshalJSONStringArray 把 []string 编码为 JSON 字符串；nil 编码为 "[]"。
func marshalJSONStringArray(arr []string) string {
	if arr == nil {
		return "[]"
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(b)
}
