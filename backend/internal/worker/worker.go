package worker

import (
	"context"
	"time"

	"ops-system/backend/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager 管理后台定时任务。
type Manager struct {
	log    *zap.Logger
	cancel context.CancelFunc
}

func NewManager(log *zap.Logger) *Manager {
	return &Manager{log: log}
}

// StartInstanceSync 启动实例状态同步（每 60 秒检查 creating/updating 状态的实例）。
func (m *Manager) StartInstanceSync(ctx context.Context, db *gorm.DB) {
	ctx, m.cancel = context.WithCancel(ctx)
	go m.instanceStatusLoop(ctx, db)
	m.log.Info("worker_instance_sync_started")
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.log.Info("worker_stopped")
}

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
			if err := instRepo.UpdateStatus(ctx, inst.ID, "running"); err != nil {
				m.log.Error("worker_instance_update_error", zap.String("id", inst.ID.String()), zap.Error(err))
				continue
			}
			m.log.Info("worker_instance_status_updated",
				zap.String("id", inst.ID.String()),
				zap.String("old_status", status),
				zap.String("new_status", "running"),
			)
		}
	}
}
