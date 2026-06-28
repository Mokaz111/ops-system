package service

import (
	"encoding/json"
	"errors"
)

// ErrInvalidPagination 分页参数错误（所有 service 共用）。
var ErrInvalidPagination = errors.New("invalid page or page_size")

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
