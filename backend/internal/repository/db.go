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
		"uni_ops_tenants_dept_id",
		"uni_ops_tenants_vm_user_id",
		"uni_ops_departments_tenant_id",
		"uni_ops_users_username",
		"uni_ops_clusters_name",
	}
	for _, name := range legacy {
		if err := db.Exec("DROP INDEX IF EXISTS " + name).Error; err != nil {
			return fmt.Errorf("drop legacy index %s: %w", name, err)
		}
	}
	return nil
}

// AutoMigrate 自动迁移元数据表。
func AutoMigrate(db *gorm.DB) error {
	if err := dropLegacyUniqueIndexes(db); err != nil {
		return err
	}
	return db.AutoMigrate(
		&model.Department{},
		&model.Tenant{},
		&model.User{},
		&model.Instance{},
		&model.PlatformScaleAudit{},
		&model.LogInstance{},
		&model.IntegrationTemplate{},
		&model.IntegrationTemplateVersion{},
		&model.IntegrationInstallation{},
		&model.IntegrationInstallationRevision{},
		&model.Metric{},
		&model.MetricTemplateMapping{},
		&model.GrafanaHost{},
		&model.Cluster{},
		&model.ScaleEvent{},
		&model.AlertRule{},
		&model.AlertEvent{},
		&model.NotificationChannel{},
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
