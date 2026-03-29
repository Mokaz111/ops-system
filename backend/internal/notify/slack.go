package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SlackSender Slack Webhook 通知。
type SlackSender struct {
	client *http.Client
}

// NewSlackSender 创建 Slack 发送器。
func NewSlackSender() *SlackSender {
	return &SlackSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SlackSender) Type() string { return "slack" }

// Send POST JSON 到 Slack Incoming Webhook。URL 从 Recipients[0] 获取。
func (s *SlackSender) Send(ctx context.Context, msg *NotifyMessage) error {
	if len(msg.Recipients) == 0 {
		return fmt.Errorf("slack: no webhook url in recipients")
	}
	webhookURL := msg.Recipients[0]

	text := fmt.Sprintf("【%s】%s\n%s", msg.Level, msg.RuleName, msg.Content)
	if msg.Details != "" {
		text += fmt.Sprintf("\n详情: %s", msg.Details)
	}

	payload := map[string]string{"text": text}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack http %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
