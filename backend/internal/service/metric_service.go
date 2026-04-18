package service

import (
	"context"
	"encoding/json"
	"errors"

	"ops-system/backend/internal/integration"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// 指标库相关业务错误。
var (
	ErrMetricNotFound   = errors.New("metric not found")
	ErrMetricNameExists = errors.New("metric name already exists")
)

// MetricService 指标库业务。
// M2：支持基于模版版本 spec 的再解析。
type MetricService struct {
	repo         *repository.MetricRepository
	templateRepo *repository.IntegrationTemplateRepository
}

func NewMetricService(
	repo *repository.MetricRepository,
	templateRepo *repository.IntegrationTemplateRepository,
) *MetricService {
	return &MetricService{repo: repo, templateRepo: templateRepo}
}

// CreateMetricRequest 创建指标。
type CreateMetricRequest struct {
	Name          string   `json:"name" binding:"required"`
	MetricType    string   `json:"metric_type"`
	Unit          string   `json:"unit"`
	Component     string   `json:"component"`
	DescriptionCN string   `json:"description_cn"`
	DescriptionEN string   `json:"description_en"`
	Labels        string   `json:"labels"`
	Examples      string   `json:"examples"`
	Tags          []string `json:"tags"`
}

// Create 创建指标条目。
func (s *MetricService) Create(ctx context.Context, req *CreateMetricRequest) (*model.Metric, error) {
	if req.Name == "" {
		return nil, errors.New("name required")
	}
	exist, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		return nil, ErrMetricNameExists
	}
	m := &model.Metric{
		Name:           req.Name,
		MetricType:     req.MetricType,
		Unit:           req.Unit,
		Component:      req.Component,
		DescriptionCN:  req.DescriptionCN,
		DescriptionEN:  req.DescriptionEN,
		Labels:         req.Labels,
		Examples:       req.Examples,
		Tags:           marshalJSONStringArray(req.Tags),
		ManualOverride: true,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get 查询。
func (s *MetricService) Get(ctx context.Context, id uuid.UUID) (*model.Metric, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

// UpdateMetricRequest 更新。
type UpdateMetricRequest struct {
	MetricType    string   `json:"metric_type"`
	Unit          string   `json:"unit"`
	Component     string   `json:"component"`
	DescriptionCN string   `json:"description_cn"`
	DescriptionEN string   `json:"description_en"`
	Labels        string   `json:"labels"`
	Examples      string   `json:"examples"`
	Tags          []string `json:"tags"`
}

// Update 更新指标。
func (s *MetricService) Update(ctx context.Context, id uuid.UUID, req *UpdateMetricRequest) (*model.Metric, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMetricNotFound
	}
	if req.MetricType != "" {
		m.MetricType = req.MetricType
	}
	if req.Unit != "" {
		m.Unit = req.Unit
	}
	if req.Component != "" {
		m.Component = req.Component
	}
	if req.DescriptionCN != "" {
		m.DescriptionCN = req.DescriptionCN
	}
	if req.DescriptionEN != "" {
		m.DescriptionEN = req.DescriptionEN
	}
	if req.Labels != "" {
		m.Labels = req.Labels
	}
	if req.Examples != "" {
		m.Examples = req.Examples
	}
	if req.Tags != nil {
		m.Tags = marshalJSONStringArray(req.Tags)
	}
	m.ManualOverride = true
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete 删除。
func (s *MetricService) Delete(ctx context.Context, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrMetricNotFound
	}
	return s.repo.Delete(ctx, id)
}

// List 分页列表。
func (s *MetricService) List(ctx context.Context, component string, templateID *uuid.UUID, keyword string, page, pageSize int) ([]model.Metric, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, repository.MetricListFilter{
		Component:  component,
		TemplateID: templateID,
		Keyword:    keyword,
		Offset:     offset,
		Limit:      pageSize,
	})
}

// Related 查询指标的关联模版 / 大盘信息。
func (s *MetricService) Related(ctx context.Context, id uuid.UUID) ([]model.MetricTemplateMapping, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMetricNotFound
	}
	return s.repo.ListMappingsByMetric(ctx, id)
}

// ReparseResult 再解析结果统计。
type ReparseResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Mappings int `json:"mappings"`
}

// Reparse 依据模版最新版本重新解析指标并 upsert。
// 如传入 version 为空则使用 template.LatestVersion。
func (s *MetricService) Reparse(ctx context.Context, templateID uuid.UUID, version string) (*ReparseResult, error) {
	if s.templateRepo == nil {
		return nil, errors.New("template repository not wired")
	}
	tpl, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, ErrIntegrationTemplateNotFound
	}
	if version == "" {
		version = tpl.LatestVersion
	}
	if version == "" {
		return nil, errors.New("template has no version")
	}
	ver, err := s.templateRepo.GetVersion(ctx, templateID, version)
	if err != nil {
		return nil, err
	}
	if ver == nil {
		return nil, ErrIntegrationVersionNotFound
	}
	spec, err := integration.ParseSpec(ver.CollectorSpec, ver.AlertSpec, ver.DashboardSpec, ver.Variables)
	if err != nil {
		return nil, err
	}
	extracted := integration.ExtractFromSpec(spec)

	res := &ReparseResult{}
	for name, info := range extracted {
		metric, err := s.repo.GetByName(ctx, name)
		if err != nil {
			return nil, err
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
			if err := s.repo.Create(ctx, metric); err != nil {
				return nil, err
			}
			res.Inserted++
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
			metric.SourceTemplateID = &tpl.ID
			metric.SourceTemplateVersion = ver.Version
			if err := s.repo.Update(ctx, metric); err != nil {
				return nil, err
			}
			res.Updated++
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
		if err := s.repo.UpsertMapping(ctx, mapping); err != nil {
			return nil, err
		}
		res.Mappings++
	}
	return res, nil
}
