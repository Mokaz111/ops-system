package log

import (
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
)

// PoolEndpoints 共享日志池端点。
type PoolEndpoints struct {
	Namespace    string
	ReleaseName  string
	InsertURL    string
	SelectURL    string
	KafkaBrokers string
	KafkaTopic   string
}

// BuildSharedPoolEndpoints 根据 Helm release 构造 Zone 日志池 URL。
func BuildSharedPoolEndpoints(namespace, releaseName, zoneSlug, kafkaBrokersOverride string) PoolEndpoints {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "logging"
	}
	release := strings.TrimSpace(releaseName)
	if release == "" {
		release = "vl-shared-stack"
	}
	slug := strings.TrimSpace(zoneSlug)
	if slug == "" {
		slug = "default"
	}
	topic := fmt.Sprintf("ops-logs-%s", slug)
	brokers := strings.TrimSpace(kafkaBrokersOverride)
	if brokers == "" {
		brokers = fmt.Sprintf("%s-kafka.%s.svc:9092", "log-kafka", ns)
	}
	return PoolEndpoints{
		Namespace:    ns,
		ReleaseName:  release,
		InsertURL:    fmt.Sprintf("http://%s-vlinsert.%s.svc:9481/insert/jsonline", release, ns),
		SelectURL:    fmt.Sprintf("http://%s-vlselect.%s.svc:9471", release, ns),
		KafkaBrokers: brokers,
		KafkaTopic:   topic,
	}
}

// PoolFromConfig 从配置构造默认端点。
func PoolFromConfig(cfg *config.LogsConfig, zoneSlug string) PoolEndpoints {
	ns := "logging"
	release := "vl-shared-stack"
	kafkaBrokers := ""
	topicPrefix := "ops-logs"
	if cfg != nil {
		if strings.TrimSpace(cfg.PoolNamespace) != "" {
			ns = strings.TrimSpace(cfg.PoolNamespace)
		}
		if strings.TrimSpace(cfg.PoolRelease) != "" {
			release = strings.TrimSpace(cfg.PoolRelease)
		}
		if strings.TrimSpace(cfg.KafkaBrokers) != "" {
			kafkaBrokers = strings.TrimSpace(cfg.KafkaBrokers)
		}
		if strings.TrimSpace(cfg.KafkaTopicPrefix) != "" {
			topicPrefix = strings.TrimSpace(cfg.KafkaTopicPrefix)
		}
	}
	ep := BuildSharedPoolEndpoints(ns, release, zoneSlug, kafkaBrokers)
	if topicPrefix != "ops-logs" {
		slug := strings.TrimSpace(zoneSlug)
		if slug == "" {
			slug = "default"
		}
		ep.KafkaTopic = fmt.Sprintf("%s-%s", topicPrefix, slug)
	}
	return ep
}
