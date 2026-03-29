package service

import (
	"context"
	"errors"
	"time"

	"ops-system/backend/internal/model"
	"ops-system/backend/internal/n9e"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrAlertEventNotFound = errors.New("alert event not found")
	ErrEventAlreadyAcked  = errors.New("event already acknowledged")
)

// NotificationSender 通知发送接口。
type NotificationSender interface {
	SendAlert(ctx context.Context, event *model.AlertEvent, channels []*model.NotificationChannel) error
}

// AlertSummary 告警汇总。
type AlertSummary struct {
	Firing       int64 `json:"firing"`
	Acknowledged int64 `json:"acknowledged"`
	Resolved     int64 `json:"resolved"`
	Total        int64 `json:"total"`
}

// AlertEventService 告警事件业务。
type AlertEventService struct {
	eventRepo   *repository.AlertEventRepository
	ruleRepo    *repository.AlertRuleRepository
	channelRepo *repository.NotificationChannelRepository
	n9e         *n9e.Client
	notifySvc   NotificationSender
	log         *zap.Logger
}

func NewAlertEventService(
	eventRepo *repository.AlertEventRepository,
	ruleRepo *repository.AlertRuleRepository,
	channelRepo *repository.NotificationChannelRepository,
	n9eClient *n9e.Client,
	notifySvc NotificationSender,
	log *zap.Logger,
) *AlertEventService {
	return &AlertEventService{
		eventRepo:   eventRepo,
		ruleRepo:    ruleRepo,
		channelRepo: channelRepo,
		n9e:         n9eClient,
		notifySvc:   notifySvc,
		log:         log,
	}
}

// ListEvents 分页列表。
func (s *AlertEventService) ListEvents(ctx context.Context, page, pageSize int, tenantID, ruleID *uuid.UUID, level, status string, startTime, endTime *time.Time) ([]model.AlertEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		return nil, 0, ErrInvalidPagination
	}
	offset := (page - 1) * pageSize
	return s.eventRepo.List(ctx, repository.AlertEventListFilter{
		TenantID:  tenantID,
		RuleID:    ruleID,
		Level:     level,
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Offset:    offset,
		Limit:     pageSize,
	})
}

// GetEvent 获取告警事件。
func (s *AlertEventService) GetEvent(ctx context.Context, id uuid.UUID) (*model.AlertEvent, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrAlertEventNotFound
	}
	return event, nil
}

// AckEvent 确认告警事件。
func (s *AlertEventService) AckEvent(ctx context.Context, eventID, userID uuid.UUID) (*model.AlertEvent, error) {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrAlertEventNotFound
	}
	if event.Status == "acknowledged" {
		return nil, ErrEventAlreadyAcked
	}

	now := time.Now()
	event.Status = "acknowledged"
	event.AckedBy = &userID
	event.AckedAt = &now
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

// Summary 告警汇总。
func (s *AlertEventService) Summary(ctx context.Context, tenantID uuid.UUID) (*AlertSummary, error) {
	firing, err := s.eventRepo.CountByStatus(ctx, tenantID, "firing")
	if err != nil {
		return nil, err
	}
	acked, err := s.eventRepo.CountByStatus(ctx, tenantID, "acknowledged")
	if err != nil {
		return nil, err
	}
	resolved, err := s.eventRepo.CountByStatus(ctx, tenantID, "resolved")
	if err != nil {
		return nil, err
	}
	return &AlertSummary{
		Firing:       firing,
		Acknowledged: acked,
		Resolved:     resolved,
		Total:        firing + acked + resolved,
	}, nil
}

// StatsByLevel 按级别统计。
func (s *AlertEventService) StatsByLevel(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]repository.LevelStats, error) {
	return s.eventRepo.StatsByLevel(ctx, tenantID, start, end)
}

// StatsByRule 按规则统计。
func (s *AlertEventService) StatsByRule(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit int) ([]repository.RuleStats, error) {
	return s.eventRepo.StatsByRule(ctx, tenantID, start, end, limit)
}

// Trend 趋势数据。
func (s *AlertEventService) Trend(ctx context.Context, tenantID uuid.UUID, start, end time.Time, interval string) ([]repository.TrendPoint, error) {
	return s.eventRepo.Trend(ctx, tenantID, start, end, interval)
}
