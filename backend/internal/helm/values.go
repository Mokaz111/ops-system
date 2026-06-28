package helm

import (
	"embed"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed values/*.yaml
var valuesFS embed.FS

// LoadValuesYAML 读取内嵌 values 文件并解析为 map（供 Helm --set 风格合并）。
// 加载后对原始文本做环境变量替换（os.ExpandEnv），使 ${GRAFANA_ADMIN_PASSWORD}
// 等占位符在交付 Helm 前被真实值替换。
func LoadValuesYAML(filename string) (map[string]interface{}, error) {
	b, err := valuesFS.ReadFile("values/" + filename)
	if err != nil {
		return nil, err
	}
	expanded := os.ExpandEnv(string(b))
	var out map[string]interface{}
	if err := yaml.Unmarshal([]byte(expanded), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}
