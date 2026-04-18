package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ops-system/backend/internal/integration"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IntegrationSeeder 幂等种子：按 template name 判断存在与否；
// 同时负责解析 spec 并 upsert ops_metrics / ops_metric_template_mappings。
type IntegrationSeeder struct {
	db           *gorm.DB
	templateRepo *repository.IntegrationTemplateRepository
	metricRepo   *repository.MetricRepository
	log          *zap.Logger
}

// NewIntegrationSeeder 构造。
func NewIntegrationSeeder(
	db *gorm.DB,
	templateRepo *repository.IntegrationTemplateRepository,
	metricRepo *repository.MetricRepository,
	log *zap.Logger,
) *IntegrationSeeder {
	return &IntegrationSeeder{db: db, templateRepo: templateRepo, metricRepo: metricRepo, log: log}
}

// SeedBuiltin 注入内置模版（幂等）。
func (s *IntegrationSeeder) SeedBuiltin(ctx context.Context) error {
	for _, seed := range integration.SeedTemplates() {
		if err := s.seedOne(ctx, seed); err != nil {
			return fmt.Errorf("seed %s: %w", seed.Name, err)
		}
	}
	return nil
}

func (s *IntegrationSeeder) seedOne(ctx context.Context, seed integration.SeededTemplate) error {
	exist, err := s.templateRepo.GetByName(ctx, seed.Name)
	if err != nil {
		return err
	}
	var tpl *model.IntegrationTemplate
	if exist != nil {
		tpl = exist
	} else {
		tpl = &model.IntegrationTemplate{
			Name:        seed.Name,
			DisplayName: seed.DisplayName,
			Category:    seed.Category,
			Component:   seed.Component,
			Description: seed.Description,
			Icon:        seed.Icon,
			Tags:        marshalJSONStringArray(seed.Tags),
			Status:      "active",
			CreatedBy:   "system",
		}
		if err := s.templateRepo.Create(ctx, tpl); err != nil {
			return err
		}
		s.log.Info("integration_template_seeded", zap.String("name", tpl.Name))
	}

	// 是否已存在该版本
	existingVer, err := s.templateRepo.GetVersion(ctx, tpl.ID, seed.Version)
	if err != nil {
		return err
	}
	if existingVer != nil {
		return s.upsertMetrics(ctx, tpl, existingVer, seed.Spec)
	}

	collectorJSON, alertJSON, dashboardJSON, variablesJSON, err := marshalSpec(seed.Spec)
	if err != nil {
		return err
	}
	ver := &model.IntegrationTemplateVersion{
		TemplateID:    tpl.ID,
		Version:       seed.Version,
		CollectorSpec: collectorJSON,
		AlertSpec:     alertJSON,
		DashboardSpec: dashboardJSON,
		Variables:     variablesJSON,
		Changelog:     seed.Changelog,
	}
	if err := s.templateRepo.CreateVersion(ctx, ver); err != nil {
		return err
	}
	tpl.LatestVersion = seed.Version
	if err := s.templateRepo.Update(ctx, tpl); err != nil {
		return err
	}
	s.log.Info("integration_template_version_seeded",
		zap.String("name", tpl.Name),
		zap.String("version", ver.Version))

	return s.upsertMetrics(ctx, tpl, ver, seed.Spec)
}

// upsertMetrics 从 spec 解析指标并写 ops_metrics / mapping。
func (s *IntegrationSeeder) upsertMetrics(
	ctx context.Context,
	tpl *model.IntegrationTemplate,
	ver *model.IntegrationTemplateVersion,
	spec integration.TemplateSpec,
) error {
	extracted := integration.ExtractFromSpec(spec)
	for name, info := range extracted {
		metric, err := s.metricRepo.GetByComponentAndName(ctx, tpl.Component, name)
		if err != nil {
			return err
		}
		metricType, unit, descCN, descEN := integration.DescribeMetric(name)
		if metric == nil {
			metric = &model.Metric{
				Name:                  name,
				MetricType:            metricType,
				Unit:                  unit,
				Component:             tpl.Component,
				DescriptionCN:         descCN,
				DescriptionEN:         descEN,
				Labels:                "[]",
				Examples:              "[]",
				SourceTemplateID:      &tpl.ID,
				SourceTemplateVersion: ver.Version,
				ManualOverride:        false,
				Tags:                  "[]",
			}
			if err := s.metricRepo.Create(ctx, metric); err != nil {
				return err
			}
		} else if !metric.ManualOverride {
			if metricType != "" {
				metric.MetricType = metricType
			}
			if unit != "" {
				metric.Unit = unit
			}
			if descCN != "" {
				metric.DescriptionCN = descCN
			}
			if descEN != "" {
				metric.DescriptionEN = descEN
			}
			if metric.Component == "" {
				metric.Component = tpl.Component
			}
			metric.SourceTemplateID = &tpl.ID
			metric.SourceTemplateVersion = ver.Version
			if err := s.metricRepo.Update(ctx, metric); err != nil {
				return err
			}
		}

		panelsJSON := "[]"
		if len(info.Panels) > 0 {
			if b, err := json.Marshal(info.Panels); err == nil {
				panelsJSON = string(b)
			}
		}
		mapping := &model.MetricTemplateMapping{
			MetricID:           metric.ID,
			TemplateID:         tpl.ID,
			TemplateVersion:    ver.Version,
			AppearsInCollector: info.AppearsInCollector,
			AppearsInDashboard: info.AppearsInDashboard,
			AppearsInAlert:     info.AppearsInAlert,
			DashboardPanels:    panelsJSON,
		}
		if err := s.metricRepo.UpsertMapping(ctx, mapping); err != nil {
			return err
		}
	}
	return nil
}

func marshalSpec(spec integration.TemplateSpec) (collector, alert, dashboard, variables string, err error) {
	collectorBytes, err := json.Marshal(spec.Collector)
	if err != nil {
		return "", "", "", "", err
	}
	alertBytes, err := json.Marshal(spec.Alert)
	if err != nil {
		return "", "", "", "", err
	}
	dashboardBytes, err := json.Marshal(spec.Dashboards)
	if err != nil {
		return "", "", "", "", err
	}
	variablesBytes, err := json.Marshal(struct {
		Variables []integration.Variable `json:"variables"`
	}{Variables: spec.Variables})
	if err != nil {
		return "", "", "", "", err
	}
	return string(collectorBytes), string(alertBytes), string(dashboardBytes), string(variablesBytes), nil
}
