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

// AutoMigrate 自动迁移元数据表。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Department{},
		&model.Tenant{},
		&model.User{},
		&model.Instance{},
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
