package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger 根据 Gin 模式初始化 zap（debug 为开发编码，否则 JSON）。
func NewLogger(mode string) (*zap.Logger, error) {
	var cfg zap.Config
	if mode == "debug" || mode == "" || mode == "test" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}
	return cfg.Build()
}
