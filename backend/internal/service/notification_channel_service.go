package service

import (
	"context"
	"errors"
	"strings"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrChannelNotFound     = errors.New("notification channel not found")
	ErrChannelNameRequired = errors.New("channel_name required")
	ErrInvalidChannelType  = errors.New("invalid channel_type")
)

var allowedChannelTypes = map[string]struct{}{
	"dingtalk": {},
	"email":    {},
	"slack":    {},
	"sms":      {},
	"webhook":  {},
}

// CreateChannelRequest 创建通知渠道请求。
type CreateChannelRequest struct {
	TenantID    uuid.UUID
	ChannelName string
	ChannelType string
	Config      string
	Enabled     bool
}

// UpdateChannelRequest 更新通知渠道请求。
type UpdateChannelRequest struct {
	ChannelName string
	ChannelType string
	Config      string
	Enabled     *bool
}

// NotificationChannelService 通知渠道业务。
type NotificationChannelService struct {
	channelRepo *repository.NotificationChannelRepository
	tenantRepo  *repository.TenantRepository
	log         *zap.Logger
}

func NewNotificationChannelService(
	channelRepo *repository.NotificationChannelRepository,
	tenantRepo *repository.TenantRepository,
	log *zap.Logger,
) *NotificationChannelService {
	return &NotificationChannelService{channelRepo: channelRepo, tenantRepo: tenantRepo, log: log}
}

// Create 创建通知渠道。
func (s *NotificationChannelService) Create(ctx context.Context, req *CreateChannelRequest) (*model.NotificationChannel, error) {
	if req.ChannelName == "" {
		return nil, ErrChannelNameRequired
	}
	if !validChannelType(req.ChannelType) {
		return nil, ErrInvalidChannelType
	}

	t, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTenantNotFound
	}

	ch := &model.NotificationChannel{
		TenantID:    req.TenantID,
		ChannelName: strings.TrimSpace(req.ChannelName),
		ChannelType: req.ChannelType,
		Config:      req.Config,
		Enabled:     req.Enabled,
	}
	if err := s.channelRepo.Create(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// Get 获取通知渠道。
func (s *NotificationChannelService) Get(ctx context.Context, id uuid.UUID) (*model.NotificationChannel, error) {
	ch, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

// List 分页列表。
func (s *NotificationChannelService) List(ctx context.Context, page, pageSize int, tenantID *uuid.UUID, channelType, keyword string) ([]model.NotificationChannel, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.channelRepo.List(ctx, repository.NotificationChannelListFilter{
		TenantID:    tenantID,
		ChannelType: channelType,
		Keyword:     keyword,
		Offset:      offset,
		Limit:       pageSize,
	})
}

// Update 更新通知渠道。
func (s *NotificationChannelService) Update(ctx context.Context, id uuid.UUID, req *UpdateChannelRequest) (*model.NotificationChannel, error) {
	ch, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}

	if req.ChannelName != "" {
		ch.ChannelName = strings.TrimSpace(req.ChannelName)
	}
	if req.ChannelType != "" {
		if !validChannelType(req.ChannelType) {
			return nil, ErrInvalidChannelType
		}
		ch.ChannelType = req.ChannelType
	}
	if req.Config != "" {
		ch.Config = req.Config
	}
	if req.Enabled != nil {
		ch.Enabled = *req.Enabled
	}

	if err := s.channelRepo.Update(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// Delete 删除通知渠道。
func (s *NotificationChannelService) Delete(ctx context.Context, id uuid.UUID) error {
	ch, err := s.channelRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ch == nil {
		return ErrChannelNotFound
	}
	return s.channelRepo.Delete(ctx, id)
}

func validChannelType(s string) bool {
	_, ok := allowedChannelTypes[s]
	return ok
}
