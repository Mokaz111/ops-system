package service

import (
	"context"
	"encoding/json"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// AuditEntry 审计写入条目。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorType  string
	Action     string
	Resource   string
	ResourceID string
	Details    any
	IP         string
	UserAgent  string
	TenantID   *uuid.UUID
	Status     string
}

// AuditListFilter 审计列表筛选。
type AuditListFilter struct {
	Action   string
	Resource string
	ActorID  *uuid.UUID
	TenantID *uuid.UUID
	Page     int
	PageSize int
}

// AuditService 平台审计业务。
type AuditService struct {
	repo *repository.AuditLogRepository
}

func NewAuditService(repo *repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Record 写入一条审计日志。
func (s *AuditService) Record(ctx context.Context, entry AuditEntry) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if entry.Action == "" {
		return errors.New("audit action required")
	}
	if entry.Resource == "" {
		return errors.New("audit resource required")
	}
	status := entry.Status
	if status == "" {
		status = "success"
	}
	details := "{}"
	if entry.Details != nil {
		b, err := json.Marshal(entry.Details)
		if err != nil {
			return errors.Wrap(err, "marshal audit details")
		}
		details = string(b)
	}
	log := &model.AuditLog{
		TenantID:   entry.TenantID,
		ActorID:    entry.ActorID,
		ActorType:  entry.ActorType,
		Action:     entry.Action,
		Resource:   entry.Resource,
		ResourceID: entry.ResourceID,
		Details:    details,
		IP:         entry.IP,
		UserAgent:  entry.UserAgent,
		Status:     status,
	}
	if err := s.repo.Create(ctx, log); err != nil {
		return errors.Wrap(err, "create audit log")
	}
	return nil
}

// List 分页查询审计日志。
func (s *AuditService) List(ctx context.Context, f AuditListFilter) ([]model.AuditLog, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, errors.New("audit service not configured")
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	return s.repo.List(ctx, repository.AuditLogListFilter{
		Action:   f.Action,
		Resource: f.Resource,
		ActorID:  f.ActorID,
		TenantID: f.TenantID,
		Offset:   (page - 1) * pageSize,
		Limit:    pageSize,
	})
}
