package service

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/log"
)

const (
	defaultLogsNamespace      = "logging"
	defaultVLRelease          = "vl-shared-stack"
	defaultVLClusterChart     = "vm/victoria-logs-cluster"
	defaultKafkaRelease       = "log-kafka"
	defaultKafkaChart         = "bitnami/kafka"
	defaultAggregatorRelease  = "log-aggregator"
	defaultVectorChart        = "vector/vector"
)

type InitLogsPipelineRequest struct {
	DryRun      bool
	Namespace   string
	ReleaseName string
	ZoneSlug    string
	Values      map[string]interface{}
}

type InitLogsPipelinePlan struct {
	DryRun           bool                   `json:"dry_run"`
	Namespace        string                 `json:"namespace"`
	ReleaseName      string                 `json:"release_name"`
	VLChart          string                 `json:"vl_chart"`
	KafkaChart       string                 `json:"kafka_chart"`
	AggregatorChart  string                 `json:"aggregator_chart"`
	KafkaBrokers     string                 `json:"kafka_brokers"`
	KafkaTopic       string                 `json:"kafka_topic"`
	InsertURL        string                 `json:"insert_url"`
	SelectURL        string                 `json:"select_url"`
	Action           string                 `json:"action"`
	Values           map[string]interface{} `json:"values"`
}

type LogPipelineService struct {
	helmClient *helm.Client
	k8sClient  *k8s.Client
	logsCfg    *config.LogsConfig
	helmCharts *config.HelmCharts
}

func NewLogPipelineService(helmClient *helm.Client, k8sClient *k8s.Client, logsCfg *config.LogsConfig, charts *config.HelmCharts) *LogPipelineService {
	return &LogPipelineService{helmClient: helmClient, k8sClient: k8sClient, logsCfg: logsCfg, helmCharts: charts}
}

func (s *LogPipelineService) InitLogsPipeline(ctx context.Context, req *InitLogsPipelineRequest) (*InitLogsPipelinePlan, error) {
	if req == nil {
		req = &InitLogsPipelineRequest{}
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = defaultLogsNamespace
		if s.logsCfg != nil && strings.TrimSpace(s.logsCfg.PoolNamespace) != "" {
			ns = strings.TrimSpace(s.logsCfg.PoolNamespace)
		}
		if slug := strings.TrimSpace(req.ZoneSlug); slug != "" {
			ns = "logging-" + slug
		}
	}
	release := strings.TrimSpace(req.ReleaseName)
	if release == "" {
		release = defaultVLRelease
		if s.logsCfg != nil && strings.TrimSpace(s.logsCfg.PoolRelease) != "" {
			release = strings.TrimSpace(s.logsCfg.PoolRelease)
		}
	}
	if !isDNS1123Name(ns) || !isDNS1123Name(release) {
		return nil, ErrInvalidNamespace
	}

	kafkaBrokersOverride := ""
	if s.logsCfg != nil {
		kafkaBrokersOverride = strings.TrimSpace(s.logsCfg.KafkaBrokers)
	}
	ep := log.BuildSharedPoolEndpoints(ns, release, req.ZoneSlug, kafkaBrokersOverride)

	vlChart := defaultVLClusterChart
	if s.helmCharts != nil && strings.TrimSpace(s.helmCharts.VLCluster) != "" {
		vlChart = strings.TrimSpace(s.helmCharts.VLCluster)
	}
	kafkaChart := defaultKafkaChart
	if s.helmCharts != nil && strings.TrimSpace(s.helmCharts.Kafka) != "" {
		kafkaChart = strings.TrimSpace(s.helmCharts.Kafka)
	}
	vectorChart := defaultVectorChart
	if s.helmCharts != nil && strings.TrimSpace(s.helmCharts.Vector) != "" {
		vectorChart = strings.TrimSpace(s.helmCharts.Vector)
	}

	vlValues := map[string]interface{}{
		"vlinsert": map[string]interface{}{"enabled": true},
		"vlselect": map[string]interface{}{"enabled": true},
		"vlstorage": map[string]interface{}{
			"enabled": true,
			"retentionPeriod": "7d",
		},
	}
	if req.Values != nil {
		vlValues = helm.MergeValues(vlValues, req.Values)
	}

	plan := &InitLogsPipelinePlan{
		DryRun:          req.DryRun,
		Namespace:       ns,
		ReleaseName:     release,
		VLChart:         vlChart,
		KafkaChart:      kafkaChart,
		AggregatorChart: vectorChart,
		KafkaBrokers:    ep.KafkaBrokers,
		KafkaTopic:      ep.KafkaTopic,
		InsertURL:       ep.InsertURL,
		SelectURL:       ep.SelectURL,
		Action:          "install_or_upgrade",
		Values:          vlValues,
	}
	if req.DryRun {
		return plan, nil
	}
	if s.helmClient == nil {
		return nil, ErrHelmOperatorNotConfigured
	}

	if err := s.helmClient.InstallOrUpgrade(ctx, release, vlChart, ns, vlValues); err != nil {
		return nil, fmt.Errorf("install victoria-logs-cluster: %w", err)
	}

	if kafkaBrokersOverride == "" {
		kafkaValues := map[string]interface{}{
			"controller": map[string]interface{}{
				"replicaCount": 1,
			},
			"broker": map[string]interface{}{
				"replicaCount": 1,
			},
			"listeners": map[string]interface{}{
				"client": map[string]interface{}{
					"protocol": "PLAINTEXT",
				},
			},
		}
		if err := s.helmClient.InstallOrUpgrade(ctx, defaultKafkaRelease, kafkaChart, ns, kafkaValues); err != nil {
			return nil, fmt.Errorf("install kafka: %w", err)
		}
	}

	aggregatorValues := buildAggregatorValues(ep.KafkaBrokers, ep.KafkaTopic, ep.InsertURL)
	if err := s.helmClient.InstallOrUpgrade(ctx, defaultAggregatorRelease, vectorChart, ns, aggregatorValues); err != nil {
		return nil, fmt.Errorf("install vector aggregator: %w", err)
	}

	if s.k8sClient != nil {
		s.k8sClient.InvalidateMapperCache()
	}
	return plan, nil
}

func buildAggregatorValues(kafkaBrokers, kafkaTopic, insertURL string) map[string]interface{} {
	vectorCfg := fmt.Sprintf(`data_dir: /vector-data-dir
sources:
  kafka_in:
    type: kafka
    bootstrap_servers: "%s"
    topics: ["%s"]
    group_id: ops-aggregator
    decoding:
      codec: json
transforms:
  filter_signals:
    type: remap
    inputs: [kafka_in]
    source: |
      level = downcase(string(.level) ?? string(.severity) ?? "")
      msg = downcase(string(.message) ?? string(._msg) ?? "")
      .ops_signal = "info"
      if level == "error" || level == "fatal" || level == "critical" {
        .ops_signal = "error"
      } else if contains(msg, "panic") || contains(msg, "oom") {
        .ops_signal = "critical"
      } else if level == "warn" || level == "warning" {
        .ops_signal = "warn"
      } else if level == "debug" || level == "trace" {
        .ops_signal = "debug"
      }
      if .ops_signal == "debug" {
        abort
      }
sinks:
  vl_out:
    type: http
    inputs: [filter_signals]
    uri: "%s"
    encoding:
      codec: json
    batch:
      max_bytes: 1048576
`, kafkaBrokers, kafkaTopic, insertURL)

	return map[string]interface{}{
		"role": "Aggregator",
		"customConfig": vectorCfg,
		"replicas":     1,
	}
}
