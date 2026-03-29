package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/n9e"
	"ops-system/backend/internal/notify"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager 管理后台定时任务。
type Manager struct {
	log    *zap.Logger
	cancel context.CancelFunc
}

// NewManager 创建 worker 管理器。
func NewManager(log *zap.Logger) *Manager {
	return &Manager{log: log}
}

// Start 启动所有后台 goroutine。
func (m *Manager) Start(ctx context.Context, cfg *config.Config, db *gorm.DB, n9eClient *n9e.Client, notifySvc *notify.NotifyService) {
	ctx, m.cancel = context.WithCancel(ctx)

	go m.instanceStatusLoop(ctx, db)
	go m.alertEventSyncLoop(ctx, cfg, db, n9eClient, notifySvc)

	m.log.Info("worker_manager_started")
}

// Stop 停止所有后台 goroutine。
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.log.Info("worker_manager_stopped")
}

// instanceStatusLoop 每 60 秒检查处于 creating/updating 状态的实例，更新为 running。
func (m *Manager) instanceStatusLoop(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	instRepo := repository.NewInstanceRepository(db)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkInstanceStatus(ctx, instRepo)
		}
	}
}

func (m *Manager) checkInstanceStatus(ctx context.Context, instRepo *repository.InstanceRepository) {
	for _, status := range []string{"creating", "updating"} {
		instances, _, err := instRepo.List(ctx, repository.InstanceListFilter{
			Status: status,
			Offset: 0,
			Limit:  200,
		})
		if err != nil {
			m.log.Error("worker_instance_list_error", zap.String("status", status), zap.Error(err))
			continue
		}
		for _, inst := range instances {
			newStatus := "running"
			if inst.ReleaseName == "" {
				newStatus = "running"
			}
			if err := instRepo.UpdateStatus(ctx, inst.ID, newStatus); err != nil {
				m.log.Error("worker_instance_update_error", zap.String("id", inst.ID.String()), zap.Error(err))
				continue
			}
			m.log.Info("worker_instance_status_updated",
				zap.String("id", inst.ID.String()),
				zap.String("old_status", status),
				zap.String("new_status", newStatus),
			)
		}
	}
}

// alertEventSyncLoop 每 30 秒从 N9E 同步活跃告警事件。
func (m *Manager) alertEventSyncLoop(ctx context.Context, cfg *config.Config, db *gorm.DB, n9eClient *n9e.Client, notifySvc *notify.NotifyService) {
	if !n9eClient.Enabled() {
		m.log.Info("worker_alert_sync_disabled", zap.String("reason", "n9e not enabled"))
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	eventRepo := repository.NewAlertEventRepository(db)
	channelRepo := repository.NewNotificationChannelRepository(db)
	ruleRepo := repository.NewAlertRuleRepository(db)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.syncAlertEvents(ctx, n9eClient, eventRepo, channelRepo, ruleRepo, notifySvc)
		}
	}
}

// n9eActiveAlert N9E 返回的活跃告警结构。
type n9eActiveAlert struct {
	RuleID   int64  `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Severity int    `json:"severity"`
	Status   string `json:"status"`
	Content  string `json:"content"`
	TriggerTime int64 `json:"trigger_time"`
}

func (m *Manager) syncAlertEvents(
	ctx context.Context,
	n9eClient *n9e.Client,
	eventRepo *repository.AlertEventRepository,
	channelRepo *repository.NotificationChannelRepository,
	ruleRepo *repository.AlertRuleRepository,
	notifySvc *notify.NotifyService,
) {
	alerts, err := m.fetchN9EActiveAlerts(ctx, n9eClient)
	if err != nil {
		m.log.Error("worker_n9e_fetch_error", zap.Error(err))
		return
	}

	for _, alert := range alerts {
		rule, err := m.findRuleByN9EID(ctx, ruleRepo, alert.RuleID)
		if err != nil {
			m.log.Error("worker_find_rule_error", zap.Int64("n9e_rule_id", alert.RuleID), zap.Error(err))
			continue
		}
		if rule == nil {
			continue
		}

		level := severityToLevel(alert.Severity)
		event := &model.AlertEvent{
			TenantID:  rule.TenantID,
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Level:     level,
			Status:    "firing",
			StartTime: time.Unix(alert.TriggerTime, 0),
			Details:   alert.Content,
			Notified:  false,
		}

		if err := eventRepo.Upsert(ctx, event); err != nil {
			m.log.Error("worker_event_upsert_error", zap.Error(err))
			continue
		}

		if event.Notified {
			continue
		}

		channels := m.resolveChannels(ctx, channelRepo, rule)
		if len(channels) > 0 {
			if err := notifySvc.SendAlert(ctx, event, channels); err != nil {
				m.log.Warn("worker_notify_error", zap.Error(err))
			}
		}
	}
}

func (m *Manager) fetchN9EActiveAlerts(ctx context.Context, client *n9e.Client) ([]n9eActiveAlert, error) {
	dat, err := client.RequestJSON(ctx, "GET", "/api/n9e/alert-cur-events?limit=100", nil)
	if err != nil {
		return nil, fmt.Errorf("n9e active alerts: %w", err)
	}
	var result struct {
		List []n9eActiveAlert `json:"list"`
	}
	if err := json.Unmarshal(dat, &result); err != nil {
		var list []n9eActiveAlert
		if err2 := json.Unmarshal(dat, &list); err2 != nil {
			return nil, fmt.Errorf("n9e unmarshal: %w", err)
		}
		return list, nil
	}
	return result.List, nil
}

func (m *Manager) findRuleByN9EID(ctx context.Context, repo *repository.AlertRuleRepository, n9eRuleID int64) (*model.AlertRule, error) {
	rules, _, err := repo.List(ctx, repository.AlertRuleListFilter{Limit: 1000})
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].N9ERuleID == n9eRuleID {
			return &rules[i], nil
		}
	}
	return nil, nil
}

func (m *Manager) resolveChannels(ctx context.Context, repo *repository.NotificationChannelRepository, rule *model.AlertRule) []*model.NotificationChannel {
	if rule.Channels == "" || rule.Channels == "[]" {
		return nil
	}
	var ids []uuid.UUID
	if err := json.Unmarshal([]byte(rule.Channels), &ids); err != nil {
		m.log.Warn("worker_parse_channel_ids_error", zap.Error(err))
		return nil
	}
	if len(ids) == 0 {
		return nil
	}

	channels, err := repo.GetByIDs(ctx, ids)
	if err != nil {
		m.log.Warn("worker_get_channels_error", zap.Error(err))
		return nil
	}

	result := make([]*model.NotificationChannel, 0, len(channels))
	for i := range channels {
		result = append(result, &channels[i])
	}
	return result
}

func severityToLevel(severity int) string {
	switch severity {
	case 1:
		return "critical"
	case 2:
		return "warning"
	default:
		return "info"
	}
}
