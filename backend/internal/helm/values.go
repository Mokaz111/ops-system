package helm

import (
	"embed"

	"gopkg.in/yaml.v3"
)

//go:embed values/*.yaml
var valuesFS embed.FS

// LoadValuesYAML 读取内嵌 values 文件并解析为 map（供 Helm --set 风格合并）。
func LoadValuesYAML(filename string) (map[string]interface{}, error) {
	b, err := valuesFS.ReadFile("values/" + filename)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}
