package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ops-system/backend/internal/config"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server HTTP 服务封装。
type Server struct {
	httpServer *http.Server
	log        *zap.Logger
}

// New 创建 HTTP 服务。
func New(cfg *config.Config, log *zap.Logger, db *gorm.DB) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	handler := NewRouter(cfg, log, db)
	s := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return &Server{httpServer: s, log: log}, nil
}

// Addr 监听地址。
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// ListenAndServe 阻塞运行。
func (s *Server) ListenAndServe() error {
	s.log.Info("http_server_starting", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
