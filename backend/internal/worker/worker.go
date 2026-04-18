package worker

import (
	"context"
	"time"

	"ops-system/backend/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Options 控制 Manager 的行为。
type Options struct {
	// InstanceStatusAutoAdvance 为 true 时把 creating/updating 状态的实例直接推进到 running。
	// 当前没有真正的 helm / k8s 健康检查，开启会产生"伪 running"，默认应保持 false。
	InstanceStatusAutoAdvance bool
}

// Manager 管理后台定时任务。
type Manager struct {
	log    *zap.Logger
	cancel context.CancelFunc
	opts   Options
}

// NewManager 构造一个未启动的 worker Manager。
func NewManager(log *zap.Logger, opts Options) *Manager {
	if log == nil {
		log = zap.NewNop()
	}
	return &Manager{log: log, opts: opts}
}

// StartInstanceSync 启动实例状态同步循环。未开启自动推进时仅记录日志，不触发 DB 写。
func (m *Manager) StartInstanceSync(ctx context.Context, db *gorm.DB) {
	ctx, m.cancel = context.WithCancel(ctx)
	go m.instanceStatusLoop(ctx, db)
	m.log.Info("worker_instance_sync_started",
		zap.Bool("auto_advance", m.opts.InstanceStatusAutoAdvance))
}

// Stop 取消后台循环。
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
		if len(instances) == 0 {
			continue
		}
		if !m.opts.InstanceStatusAutoAdvance {
			m.log.Debug("worker_instance_pending_noop",
				zap.String("status", status),
				zap.Int("count", len(instances)),
				zap.String("hint", "enable worker.instance_status_auto_advance to auto-promote"))
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
