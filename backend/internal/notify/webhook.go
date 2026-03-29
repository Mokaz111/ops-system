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

// WebhookSender 通用 Webhook 通知。通过 NotifyMessage 中的 Details 字段携带的 URL 发送。
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender 创建 Webhook 发送器。
func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *WebhookSender) Type() string { return "webhook" }

// webhookPayload Webhook 请求体。
type webhookPayload struct {
	RuleName  string `json:"rule_name"`
	Level     string `json:"level"`
	Status    string `json:"status"`
	Content   string `json:"content"`
	StartTime string `json:"start_time"`
	Details   string `json:"details"`
}

// Send POST JSON 到 Webhook URL。URL 从 Recipients[0] 中获取（由 SendAlert 解析 channel config 注入）。
func (w *WebhookSender) Send(ctx context.Context, msg *NotifyMessage) error {
	if len(msg.Recipients) == 0 {
		return fmt.Errorf("webhook: no url in recipients")
	}
	webhookURL := msg.Recipients[0]

	payload := webhookPayload{
		RuleName:  msg.RuleName,
		Level:     msg.Level,
		Status:    msg.Status,
		Content:   msg.Content,
		StartTime: msg.StartTime.Format(time.RFC3339),
		Details:   msg.Details,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook http %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
