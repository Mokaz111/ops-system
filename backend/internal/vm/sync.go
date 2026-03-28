package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"

	"go.uber.org/zap"
)

// SyncService 租户与 vmauth / 外部 Webhook 的同步（§2.3）。
type SyncService struct {
	cfg    *config.VMConfig
	log    *zap.Logger
	client *http.Client
}

// NewSyncService 创建同步服务；cfg 可为 nil（按零值处理）。
func NewSyncService(cfg *config.VMConfig, log *zap.Logger) *SyncService {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg == nil {
		cfg = &config.VMConfig{}
	}
	sec := cfg.HTTPTimeoutSeconds
	if sec <= 0 {
		sec = 15
	}
	return &SyncService{
		cfg: cfg,
		log: log,
		client: &http.Client{
			Timeout: time.Duration(sec) * time.Second,
		},
	}
}

// InsertURL 返回对外写入地址（展示用）。
func (s *SyncService) InsertURL(vmuserID string) string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return InsertURL(s.cfg.VMAuthBaseURL, vmuserID)
}

// syncPayload Webhook 负载（便于侧车或自定义服务注册 vmuser）。
type syncPayload struct {
	Event       string `json:"event"`
	TenantID    string `json:"tenant_id"`
	TenantName  string `json:"tenant_name"`
	VMUserID    string `json:"vmuser_id"`
	VMUserKey   string `json:"vmuser_key,omitempty"`
	QuotaConfig string `json:"quota_config,omitempty"`
}

// OnTenantCreated 租户落库成功后调用（失败仅打日志，不回滚租户）。
func (s *SyncService) OnTenantCreated(ctx context.Context, t *model.Tenant) {
	if s == nil || t == nil {
		return
	}
	if !s.cfg.SyncEnabled || s.cfg.VMAuthWebhookURL == "" {
		return
	}
	body := syncPayload{
		Event:       "tenant.created",
		TenantID:    t.ID.String(),
		TenantName:  t.TenantName,
		VMUserID:    t.VMUserID,
		VMUserKey:   t.VMUserKey,
		QuotaConfig: t.QuotaConfig,
	}
	if err := s.postWebhook(ctx, body); err != nil {
		s.log.Warn("vm_sync_tenant_created_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
		return
	}
	s.log.Info("vm_sync_tenant_created_ok", zap.String("tenant_id", t.ID.String()), zap.String("vmuser_id", t.VMUserID))
}

// OnTenantDeleted 删除租户前调用。
func (s *SyncService) OnTenantDeleted(ctx context.Context, t *model.Tenant) {
	if s == nil || t == nil {
		return
	}
	if !s.cfg.SyncEnabled || s.cfg.VMAuthWebhookURL == "" {
		return
	}
	body := syncPayload{
		Event:      "tenant.deleted",
		TenantID:   t.ID.String(),
		TenantName: t.TenantName,
		VMUserID:   t.VMUserID,
	}
	if err := s.postWebhook(ctx, body); err != nil {
		s.log.Warn("vm_sync_tenant_deleted_failed", zap.String("tenant_id", t.ID.String()), zap.Error(err))
		return
	}
	s.log.Info("vm_sync_tenant_deleted_ok", zap.String("tenant_id", t.ID.String()), zap.String("vmuser_id", t.VMUserID))
}

func (s *SyncService) postWebhook(ctx context.Context, body syncPayload) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.VMAuthWebhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}
