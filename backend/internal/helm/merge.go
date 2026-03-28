package helm

// MergeValues 浅层合并：嵌套 map 递归合并，其它键后者覆盖前者。
func MergeValues(base, overlay map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		if b, ok := out[k]; ok {
			if bm, ok := b.(map[string]interface{}); ok {
				if vm, ok := v.(map[string]interface{}); ok {
					out[k] = MergeValues(bm, vm)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
