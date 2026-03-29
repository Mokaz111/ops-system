package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/server"
	"ops-system/backend/internal/worker"
	"ops-system/backend/pkg/utils"

	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to config yaml (default: ./configs/config.yaml)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	log, err := utils.NewLogger(cfg.Server.Mode)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	db, err := repository.NewPostgres(cfg, log)
	if err != nil {
		log.Fatal("postgres_init", zap.Error(err))
	}
	defer func() {
		if err := repository.Close(db); err != nil {
			log.Error("postgres_close", zap.Error(err))
		}
	}()

	if err := repository.AutoMigrate(db); err != nil {
		log.Fatal("auto_migrate", zap.Error(err))
	}
	log.Info("db_migrate_ok")

	wm := worker.NewManager(log)
	wm.StartInstanceSync(context.Background(), db)
	defer wm.Stop()

	srv, err := server.New(cfg, log, db)
	if err != nil {
		log.Fatal(err.Error())
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server_shutdown", zap.Error(err))
	}
	log.Info("server_stopped")
}
