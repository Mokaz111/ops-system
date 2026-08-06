package repository

import (
	"fmt"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewPostgres 初始化 GORM 并配置连接池。
func NewPostgres(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	d := cfg.Database
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)

	level := gormlogger.Warn
	switch cfg.Server.Mode {
	case "debug", "test", "":
		level = gormlogger.Info
	}

	gcfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
	}

	db, err := gorm.Open(postgres.Open(dsn), gcfg)
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(d.MaxOpenConns)
	sqlDB.SetMaxIdleConns(d.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(d.ConnMaxLifetimeMinutes) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	log.Info("postgres_connected",
		zap.String("host", d.Host),
		zap.Int("port", d.Port),
		zap.String("dbname", d.Name),
		zap.Int("max_open_conns", d.MaxOpenConns),
		zap.Int("max_idle_conns", d.MaxIdleConns),
		zap.Int("conn_max_lifetime_minutes", d.ConnMaxLifetimeMinutes),
	)

	return db, nil
}

// dropLegacyUniqueIndexes 在 AutoMigrate 之前清理已废弃的非 partial 唯一索引。
//
// 这些索引在 Stage 2 / Stage 4 的修复里被改成了 partial unique（WHERE deleted_at IS NULL），
// 新索引使用新的命名前缀（`uk_*`），但老的 GORM 默认名（`uni_<table>_<col>`）不会被
// AutoMigrate 自动删除；若遗留下来会导致"软删除后同名重建"触发 UNIQUE 冲突。
// 这里用 DROP INDEX IF EXISTS 让执行幂等（postgres 方言），首次升级后变成 no-op。
func dropLegacyUniqueIndexes(db *gorm.DB) error {
	legacy := []string{
		"uni_ops_workspaces_vm_user_id",
		"uni_ops_users_username",
		"uni_ops_clusters_name",
		"uni_ops_integration_templates_name",
	}
	for _, name := range legacy {
		if err := db.Exec("DROP INDEX IF EXISTS " + name).Error; err != nil {
			return fmt.Errorf("drop legacy index %s: %w", name, err)
		}
	}
	return nil
}

// dropLegacyTables 删除已废弃功能遗留的表（开发阶段无向前兼容要求，直接 DROP）。
func dropLegacyTables(db *gorm.DB) error {
	legacy := []string{
		"ops_collector_targets",
		"ops_recording_rules",
		"ops_alert_receivers",
		"ops_scale_events",
		"ops_platform_scale_audits",
	}
	for _, name := range legacy {
		if err := db.Exec("DROP TABLE IF EXISTS " + name).Error; err != nil {
			return fmt.Errorf("drop legacy table %s: %w", name, err)
		}
	}
	return nil
}

// dropLegacyColumns 删除已废弃列。
func dropLegacyColumns(db *gorm.DB) error {
	legacy := []struct {
		table  string
		column string
	}{
		{"ops_workspaces", "n9e_team_id"},
		{"ops_users", "workspace_id"},
	}
	for _, item := range legacy {
		if err := db.Exec("ALTER TABLE "+item.table+" DROP COLUMN IF EXISTS "+item.column).Error; err != nil {
			return fmt.Errorf("drop legacy column %s.%s: %w", item.table, item.column, err)
		}
	}
	return nil
}

// AutoMigrate 自动迁移元数据表。
func AutoMigrate(db *gorm.DB) error {
	if err := dropLegacyUniqueIndexes(db); err != nil {
		return err
	}
	if err := dropLegacyTables(db); err != nil {
		return err
	}
	if err := dropLegacyColumns(db); err != nil {
		return err
	}
	return db.AutoMigrate(
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.User{},
		&model.APIToken{},
		&model.VMCluster{},
		&model.VMRoute{},
		&model.Datasource{},
		&model.ProvisioningTask{},
		&model.AuditLog{},
		&model.Instance{},
		&model.LogInstance{},
		&model.LogCluster{},
		&model.IntegrationTemplate{},
		&model.IntegrationTemplateVersion{},
		&model.IntegrationInstallation{},
		&model.IntegrationInstallationRevision{},
		&model.Metric{},
		&model.MetricTemplateMapping{},
		&model.GrafanaInstance{},
		&model.Cluster{},
		&model.AlertRule{},
		&model.AlertEvent{},
		&model.NotificationChannel{},
		&model.Zone{},
		&model.BusinessCluster{},
		&model.Entity{},
		&model.MetricSet{},
		&model.LogSet{},
		&model.DataLink{},
	)
}

// Close 关闭底层连接池。
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
