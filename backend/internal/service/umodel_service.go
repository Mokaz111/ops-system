package service

import (
	"context"
	"encoding/json"
	"errors"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrEntityNotFound     = errors.New("entity not found")
	ErrMetricSetNotFound  = errors.New("metric set not found")
	ErrLogSetNotFound     = errors.New("log set not found")
	ErrDataLinkNotFound   = errors.New("data link not found")
	ErrUModelNameRequired = errors.New("name required")
)

type UModelService struct {
	entities   *repository.EntityRepository
	metricSets *repository.MetricSetRepository
	logSets    *repository.LogSetRepository
	dataLinks  *repository.DataLinkRepository
}

func NewUModelService(
	entities *repository.EntityRepository,
	metricSets *repository.MetricSetRepository,
	logSets *repository.LogSetRepository,
	dataLinks *repository.DataLinkRepository,
) *UModelService {
	return &UModelService{entities: entities, metricSets: metricSets, logSets: logSets, dataLinks: dataLinks}
}

type CreateEntityRequest struct {
	EntityType  string            `json:"entity_type" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	DisplayName string            `json:"display_name"`
	Labels      map[string]string `json:"labels"`
}

func (s *UModelService) CreateEntity(ctx context.Context, tenantID uuid.UUID, req *CreateEntityRequest) (*model.Entity, error) {
	if req.Name == "" {
		return nil, ErrUModelNameRequired
	}
	labels := marshalJSON(req.Labels)
	e := &model.Entity{
		TenantID:    tenantID,
		EntityType:  req.EntityType,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Status:      "active",
		Labels:      labels,
	}
	if err := s.entities.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *UModelService) ListEntities(ctx context.Context, tenantID uuid.UUID, entityType, keyword string, page, pageSize int) ([]model.Entity, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.entities.List(ctx, repository.EntityListFilter{
		TenantID: tenantID, EntityType: entityType, Keyword: keyword,
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
}

func (s *UModelService) GetEntity(ctx context.Context, tenantID, id uuid.UUID) (*model.Entity, error) {
	e, err := s.entities.GetByID(ctx, id)
	if err != nil || e == nil || e.TenantID != tenantID {
		return nil, ErrEntityNotFound
	}
	return e, nil
}

func (s *UModelService) DeleteEntity(ctx context.Context, tenantID, id uuid.UUID) error {
	e, err := s.GetEntity(ctx, tenantID, id)
	if err != nil {
		return err
	}
	return s.entities.Delete(ctx, e.ID)
}

type CreateMetricSetRequest struct {
	Name        string            `json:"name" binding:"required"`
	DisplayName string            `json:"display_name"`
	Component   string            `json:"component"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
}

func (s *UModelService) CreateMetricSet(ctx context.Context, tenantID uuid.UUID, req *CreateMetricSetRequest) (*model.MetricSet, error) {
	if req.Name == "" {
		return nil, ErrUModelNameRequired
	}
	m := &model.MetricSet{
		TenantID:    tenantID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Component:   req.Component,
		Description: req.Description,
		Labels:      marshalJSON(req.Labels),
		Status:      "active",
	}
	if err := s.metricSets.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *UModelService) ListMetricSets(ctx context.Context, tenantID uuid.UUID, component, keyword string, page, pageSize int) ([]model.MetricSet, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.metricSets.List(ctx, repository.MetricSetListFilter{
		TenantID: tenantID, Component: component, Keyword: keyword,
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
}

func (s *UModelService) GetMetricSet(ctx context.Context, tenantID, id uuid.UUID) (*model.MetricSet, error) {
	m, err := s.metricSets.GetByID(ctx, id)
	if err != nil || m == nil || m.TenantID != tenantID {
		return nil, ErrMetricSetNotFound
	}
	return m, nil
}

func (s *UModelService) DeleteMetricSet(ctx context.Context, tenantID, id uuid.UUID) error {
	m, err := s.GetMetricSet(ctx, tenantID, id)
	if err != nil {
		return err
	}
	return s.metricSets.Delete(ctx, m.ID)
}

type CreateLogSetRequest struct {
	Name        string            `json:"name" binding:"required"`
	DisplayName string            `json:"display_name"`
	Component   string            `json:"component"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
}

func (s *UModelService) CreateLogSet(ctx context.Context, tenantID uuid.UUID, req *CreateLogSetRequest) (*model.LogSet, error) {
	if req.Name == "" {
		return nil, ErrUModelNameRequired
	}
	l := &model.LogSet{
		TenantID:    tenantID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Component:   req.Component,
		Description: req.Description,
		Labels:      marshalJSON(req.Labels),
		Status:      "active",
	}
	if err := s.logSets.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *UModelService) ListLogSets(ctx context.Context, tenantID uuid.UUID, component, keyword string, page, pageSize int) ([]model.LogSet, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.logSets.List(ctx, repository.LogSetListFilter{
		TenantID: tenantID, Component: component, Keyword: keyword,
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
}

func (s *UModelService) GetLogSet(ctx context.Context, tenantID, id uuid.UUID) (*model.LogSet, error) {
	l, err := s.logSets.GetByID(ctx, id)
	if err != nil || l == nil || l.TenantID != tenantID {
		return nil, ErrLogSetNotFound
	}
	return l, nil
}

func (s *UModelService) DeleteLogSet(ctx context.Context, tenantID, id uuid.UUID) error {
	l, err := s.GetLogSet(ctx, tenantID, id)
	if err != nil {
		return err
	}
	return s.logSets.Delete(ctx, l.ID)
}

type CreateDataLinkRequest struct {
	EntityID     string `json:"entity_id" binding:"required"`
	TargetType   string `json:"target_type" binding:"required"`
	TargetID     string `json:"target_id" binding:"required"`
	RelationType string `json:"relation_type"`
}

func (s *UModelService) CreateDataLink(ctx context.Context, tenantID uuid.UUID, req *CreateDataLinkRequest) (*model.DataLink, error) {
	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		return nil, ErrMetricSetNotFound
	}
	if _, err := s.GetEntity(ctx, tenantID, entityID); err != nil {
		return nil, err
	}
	if req.TargetType == "metric_set" {
		if _, err := s.GetMetricSet(ctx, tenantID, targetID); err != nil {
			return nil, err
		}
	}
	if req.TargetType == "log_set" {
		if _, err := s.GetLogSet(ctx, tenantID, targetID); err != nil {
			return nil, err
		}
	}
	rel := req.RelationType
	if rel == "" {
		rel = "observes"
	}
	d := &model.DataLink{
		TenantID: tenantID, EntityID: entityID, TargetType: req.TargetType,
		TargetID: targetID, RelationType: rel, Status: "active",
	}
	if err := s.dataLinks.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *UModelService) ListDataLinks(ctx context.Context, tenantID, entityID uuid.UUID) ([]model.DataLink, error) {
	if _, err := s.GetEntity(ctx, tenantID, entityID); err != nil {
		return nil, err
	}
	return s.dataLinks.ListByEntity(ctx, tenantID, entityID)
}

func (s *UModelService) DeleteDataLink(ctx context.Context, tenantID, id uuid.UUID) error {
	d, err := s.dataLinks.GetByID(ctx, id)
	if err != nil || d == nil || d.TenantID != tenantID {
		return ErrDataLinkNotFound
	}
	return s.dataLinks.Delete(ctx, id)
}

func marshalJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
