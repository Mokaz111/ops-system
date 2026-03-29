package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ops-system/backend/internal/model"

	"go.uber.org/zap"
)

// NotifyMessage 统一通知消息体。
type NotifyMessage struct {
	RuleName   string
	Level      string
	Status     string
	Content    string
	Details    string
	StartTime  time.Time
	Recipients []string
	IsAtAll    bool
}

// Sender 发送渠道接口。
type Sender interface {
	Type() string
	Send(ctx context.Context, msg *NotifyMessage) error
}

// NotifyService 组合多个 Sender，按渠道类型分发通知。
type NotifyService struct {
	senders map[string]Sender
	log     *zap.Logger
}

// NewNotifyService 创建通知服务。
func NewNotifyService(log *zap.Logger) *NotifyService {
	return &NotifyService{
		senders: make(map[string]Sender),
		log:     log,
	}
}

// Register 注册一个发送渠道。
func (s *NotifyService) Register(sender Sender) {
	s.senders[sender.Type()] = sender
	s.log.Info("notify_sender_registered", zap.String("type", sender.Type()))
}

// SendAlert 遍历通知渠道，逐一发送告警通知。单个渠道失败只记日志，不中断其他渠道。
func (s *NotifyService) SendAlert(ctx context.Context, event *model.AlertEvent, channels []*model.NotificationChannel) error {
	msg := &NotifyMessage{
		RuleName:  event.RuleName,
		Level:     event.Level,
		Status:    event.Status,
		Content:   fmt.Sprintf("[%s] %s 告警触发", event.Level, event.RuleName),
		Details:   event.Details,
		StartTime: event.StartTime,
	}

	var lastErr error
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		sender, ok := s.senders[ch.ChannelType]
		if !ok {
			s.log.Warn("notify_sender_not_found", zap.String("channel_type", ch.ChannelType), zap.String("channel_name", ch.ChannelName))
			continue
		}

		chMsg := *msg
		var cfg map[string]any
		if ch.Config != "" {
			if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
				s.log.Warn("notify_channel_config_parse_error", zap.Error(err), zap.String("channel_name", ch.ChannelName))
				continue
			}
		}
		if recipients, ok := cfg["recipients"].([]any); ok {
			for _, r := range recipients {
				if rs, ok := r.(string); ok {
					chMsg.Recipients = append(chMsg.Recipients, rs)
				}
			}
		}
		if atAll, ok := cfg["is_at_all"].(bool); ok {
			chMsg.IsAtAll = atAll
		}

		if err := sender.Send(ctx, &chMsg); err != nil {
			s.log.Warn("notify_send_failed",
				zap.String("channel_type", ch.ChannelType),
				zap.String("channel_name", ch.ChannelName),
				zap.Error(err),
			)
			lastErr = err
			continue
		}
		s.log.Info("notify_send_ok", zap.String("channel_type", ch.ChannelType), zap.String("channel_name", ch.ChannelName))
	}
	return lastErr
}
