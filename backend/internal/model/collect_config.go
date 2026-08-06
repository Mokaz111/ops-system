package model

import (
	"encoding/json"
	"strings"
)

// MetricsCollectConfig 业务集群指标采集配置（VMAgent）。
type MetricsCollectConfig struct {
	// SelectAllByDefault 是否自动发现集群内所有 scrape 对象（默认 true）。
	SelectAllByDefault *bool `json:"select_all_by_default,omitempty"`
	// ScrapeInterval 全局抓取间隔，如 30s（默认 30s）。
	ScrapeInterval string `json:"scrape_interval,omitempty"`
	// ScrapeTimeout 抓取超时（默认 10s）。
	ScrapeTimeout string `json:"scrape_timeout,omitempty"`
	// NamespaceInclude 仅采集这些命名空间的目标；为空表示不限制。
	NamespaceInclude []string `json:"namespace_include,omitempty"`
	// NamespaceExclude 丢弃这些命名空间的指标（基于 namespace 标签）。
	NamespaceExclude []string `json:"namespace_exclude,omitempty"`
}

// LogsCollectConfig 业务集群日志采集配置（Vector）。
type LogsCollectConfig struct {
	// NamespaceInclude 仅采集这些命名空间的 Pod 日志；为空表示不限制。
	NamespaceInclude []string `json:"namespace_include,omitempty"`
	// NamespaceExclude 排除这些命名空间。
	NamespaceExclude []string `json:"namespace_exclude,omitempty"`
	// ExcludePaths 排除日志路径 glob，例如 **/exclude/**。
	ExcludePaths []string `json:"exclude_paths,omitempty"`
}

// DefaultMetricsCollectConfig 默认指标采集配置。
func DefaultMetricsCollectConfig() MetricsCollectConfig {
	t := true
	return MetricsCollectConfig{
		SelectAllByDefault: &t,
		ScrapeInterval:     "30s",
		ScrapeTimeout:      "10s",
		NamespaceInclude:   []string{},
		NamespaceExclude:   []string{},
	}
}

// DefaultLogsCollectConfig 默认日志采集配置。
func DefaultLogsCollectConfig() LogsCollectConfig {
	return LogsCollectConfig{
		NamespaceInclude: []string{},
		NamespaceExclude: []string{"kube-system"},
		ExcludePaths:     []string{},
	}
}

// ParseMetricsCollectConfig 解析 JSON；空串返回默认值。
func ParseMetricsCollectConfig(raw string) MetricsCollectConfig {
	cfg := DefaultMetricsCollectConfig()
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return cfg
	}
	var parsed MetricsCollectConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return cfg
	}
	if parsed.SelectAllByDefault != nil {
		cfg.SelectAllByDefault = parsed.SelectAllByDefault
	}
	if strings.TrimSpace(parsed.ScrapeInterval) != "" {
		cfg.ScrapeInterval = strings.TrimSpace(parsed.ScrapeInterval)
	}
	if strings.TrimSpace(parsed.ScrapeTimeout) != "" {
		cfg.ScrapeTimeout = strings.TrimSpace(parsed.ScrapeTimeout)
	}
	if parsed.NamespaceInclude != nil {
		cfg.NamespaceInclude = normalizeNSList(parsed.NamespaceInclude)
	}
	if parsed.NamespaceExclude != nil {
		cfg.NamespaceExclude = normalizeNSList(parsed.NamespaceExclude)
	}
	return cfg
}

// ParseLogsCollectConfig 解析 JSON；空串返回默认值。
func ParseLogsCollectConfig(raw string) LogsCollectConfig {
	cfg := DefaultLogsCollectConfig()
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" || raw == "null" {
		return cfg
	}
	var parsed LogsCollectConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return cfg
	}
	if parsed.NamespaceInclude != nil {
		cfg.NamespaceInclude = normalizeNSList(parsed.NamespaceInclude)
	}
	if parsed.NamespaceExclude != nil {
		cfg.NamespaceExclude = normalizeNSList(parsed.NamespaceExclude)
	}
	if parsed.ExcludePaths != nil {
		cfg.ExcludePaths = normalizeNSList(parsed.ExcludePaths)
	}
	return cfg
}

func normalizeNSList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// MarshalCollectConfig 序列化采集配置。
func MarshalCollectConfig(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
