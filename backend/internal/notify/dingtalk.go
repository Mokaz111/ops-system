package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DingtalkSender 钉钉机器人通知。
type DingtalkSender struct {
	webhookURL string
	secret     string
	client     *http.Client
}

// NewDingtalkSender 创建钉钉发送器。
func NewDingtalkSender(webhookURL, secret string) *DingtalkSender {
	return &DingtalkSender{
		webhookURL: webhookURL,
		secret:     secret,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *DingtalkSender) Type() string { return "dingtalk" }

// Send 发送钉钉 markdown 消息。
func (d *DingtalkSender) Send(ctx context.Context, msg *NotifyMessage) error {
	webhook, err := d.signURL()
	if err != nil {
		return fmt.Errorf("dingtalk sign url: %w", err)
	}

	title := fmt.Sprintf("【%s】%s", msg.Level, msg.RuleName)
	text := fmt.Sprintf("### %s\n\n**状态**: %s\n\n**时间**: %s\n\n%s",
		title, msg.Status, msg.StartTime.Format("2006-01-02 15:04:05"), msg.Content)
	if msg.Details != "" {
		text += fmt.Sprintf("\n\n**详情**: %s", msg.Details)
	}

	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}
	if msg.IsAtAll || len(msg.Recipients) > 0 {
		at := map[string]any{"isAtAll": msg.IsAtAll}
		if len(msg.Recipients) > 0 {
			at["atMobiles"] = msg.Recipients
		}
		payload["at"] = at
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk http %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (d *DingtalkSender) signURL() (string, error) {
	if d.secret == "" {
		return d.webhookURL, nil
	}
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	strToSign := ts + "\n" + d.secret

	mac := hmac.New(sha256.New, []byte(d.secret))
	mac.Write([]byte(strToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	u, err := url.Parse(d.webhookURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("timestamp", ts)
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
