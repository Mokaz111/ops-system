package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"ops-system/backend/internal/model"
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

// AlertmanagerWebhookPayload Alertmanager webhook 格式（简化）。
type AlertmanagerWebhookPayload struct {
	Status string `json:"status"`
	Alerts []struct {
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		StartsAt     time.Time         `json:"startsAt"`
		EndsAt       time.Time         `json:"endsAt"`
		Fingerprint  string            `json:"fingerprint"`
		GeneratorURL string            `json:"generatorURL"`
	} `json:"alerts"`
}

// AlertEventService 告警事件业务。
type AlertEventService struct {
	eventRepo   *repository.AlertEventRepository
	ruleRepo    *repository.AlertRuleRepository
	channelRepo *repository.NotificationChannelRepository
	notifySvc   NotificationSender
	log         *zap.Logger
}

func NewAlertEventService(
	eventRepo *repository.AlertEventRepository,
	ruleRepo *repository.AlertRuleRepository,
	channelRepo *repository.NotificationChannelRepository,
	notifySvc NotificationSender,
	log *zap.Logger,
) *AlertEventService {
	return &AlertEventService{
		eventRepo:   eventRepo,
		ruleRepo:    ruleRepo,
		channelRepo: channelRepo,
		notifySvc:   notifySvc,
		log:         log,
	}
}

// IngestAlertmanager 处理 Alertmanager webhook 回调。
func (s *AlertEventService) IngestAlertmanager(ctx context.Context, payload *AlertmanagerWebhookPayload) error {
	if s == nil || payload == nil {
		return nil
	}
	for _, alert := range payload.Alerts {
		vmRuleName := alert.Labels["vm_rule_name"]
		if vmRuleName == "" {
			vmRuleName = alert.Labels["alertname"]
		}
		tenantIDStr := alert.Labels["ops_tenant_id"]
		if tenantIDStr == "" {
			tenantIDStr = alert.Labels["tenant_id"]
		}
		if vmRuleName == "" || tenantIDStr == "" {
			s.log.Warn("alertmanager_alert_missing_labels", zap.Any("labels", alert.Labels))
			continue
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			continue
		}

		rule, err := s.ruleRepo.GetByVMRuleName(ctx, tenantID, vmRuleName)
		if err != nil || rule == nil {
			s.log.Warn("alertmanager_rule_not_found", zap.String("vm_rule_name", vmRuleName), zap.String("tenant_id", tenantIDStr))
			continue
		}

		level := alert.Labels["severity"]
		if level == "" {
			level = rule.Level
		}

		details, _ := json.Marshal(map[string]any{
			"labels":       alert.Labels,
			"annotations":  alert.Annotations,
			"fingerprint":  alert.Fingerprint,
			"generatorURL": alert.GeneratorURL,
		})

		if alert.Status == "firing" {
			event := &model.AlertEvent{
				TenantID:  tenantID,
				RuleID:    rule.ID,
				RuleName:  rule.RuleName,
				Level:     level,
				Status:    "firing",
				StartTime: alert.StartsAt,
				Details:   string(details),
			}
			if event.StartTime.IsZero() {
				event.StartTime = time.Now().UTC()
			}
			if err := s.eventRepo.Create(ctx, event); err != nil {
				s.log.Warn("alert_event_create_failed", zap.Error(err))
				continue
			}
			s.dispatchNotification(ctx, rule, event)
		} else if alert.Status == "resolved" {
			events, _, err := s.eventRepo.List(ctx, repository.AlertEventListFilter{
				TenantID: &tenantID,
				RuleID:   &rule.ID,
				Status:   "firing",
				Limit:    10,
			})
			if err != nil {
				continue
			}
			now := time.Now().UTC()
			for i := range events {
				events[i].Status = "resolved"
				events[i].EndTime = &now
				_ = s.eventRepo.Update(ctx, &events[i])
			}
		}
	}
	return nil
}

func (s *AlertEventService) dispatchNotification(ctx context.Context, rule *model.AlertRule, event *model.AlertEvent) {
	if s.notifySvc == nil || rule.Channels == "" || rule.Channels == "[]" {
		return
	}
	var channelIDs []string
	if err := json.Unmarshal([]byte(rule.Channels), &channelIDs); err != nil {
		return
	}
	var channels []*model.NotificationChannel
	for _, idStr := range channelIDs {
		id, err := uuid.Parse(strings.TrimSpace(idStr))
		if err != nil {
			continue
		}
		ch, err := s.channelRepo.GetByID(ctx, id)
		if err != nil || ch == nil || !ch.Enabled {
			continue
		}
		channels = append(channels, ch)
	}
	if len(channels) == 0 {
		return
	}
	if err := s.notifySvc.SendAlert(ctx, event, channels); err != nil {
		s.log.Warn("alert_notify_failed", zap.Error(err), zap.String("event_id", event.ID.String()))
		return
	}
	event.Notified = true
	_ = s.eventRepo.Update(ctx, event)
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
