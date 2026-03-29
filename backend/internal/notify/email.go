package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailSender 邮件通知。
type EmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewEmailSender 创建邮件发送器。
func NewEmailSender(host string, port int, username, password, from string) *EmailSender {
	return &EmailSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (e *EmailSender) Type() string { return "email" }

// Send 通过 SMTP 发送 HTML 邮件。
func (e *EmailSender) Send(_ context.Context, msg *NotifyMessage) error {
	if len(msg.Recipients) == 0 {
		return fmt.Errorf("email: no recipients")
	}

	subject := fmt.Sprintf("【%s】%s", msg.Level, msg.RuleName)
	body := fmt.Sprintf(`<html><body>
<h3>%s</h3>
<p><b>状态:</b> %s</p>
<p><b>时间:</b> %s</p>
<p>%s</p>
<p><b>详情:</b> %s</p>
</body></html>`,
		subject, msg.Status, msg.StartTime.Format("2006-01-02 15:04:05"),
		msg.Content, msg.Details)

	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		e.from, strings.Join(msg.Recipients, ","), subject)
	raw := header + body

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	return smtp.SendMail(addr, auth, e.from, msg.Recipients, []byte(raw))
}
