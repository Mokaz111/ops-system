package notify

import (
	"context"

	"go.uber.org/zap"
)

// SMSSender 短信通知（占位实现）。
type SMSSender struct {
	log *zap.Logger
}

// NewSMSSender 创建短信发送器（占位）。
func NewSMSSender(log *zap.Logger) *SMSSender {
	return &SMSSender{log: log}
}

func (s *SMSSender) Type() string { return "sms" }

// Send 占位实现，记录警告日志并返回 nil。
func (s *SMSSender) Send(_ context.Context, msg *NotifyMessage) error {
	s.log.Warn("sms_not_configured",
		zap.String("rule_name", msg.RuleName),
		zap.String("level", msg.Level),
		zap.Int("recipients", len(msg.Recipients)),
	)
	return nil
}
