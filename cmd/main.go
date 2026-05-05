package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web-manager"
	_ "web-manager/docs"
	"web-manager/internal/config"
	"web-manager/internal/envfile"
	"web-manager/internal/logging"
)

// @title Web Manager Backend API
// @version 1.0
// @description Multi-app backend with auth, MFA (TOTP), and admin APIs.
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = envfile.Load()
	cfg := config.MustLoad()
	logger := logging.New(cfg.Env)
	defer func() { _ = logger.Sync() }()

	app, cleanup, err := webmanager.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("failed to create server", logging.Err(err))
	}
	defer cleanup()

	go func() {
		addr := cfg.HTTP.Addr()
		logger.Info("http server starting", logging.Str("addr", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("http server failed", logging.Err(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_ = ctx
	if err := app.Shutdown(); err != nil {
		logger.Error("graceful shutdown failed", logging.Err(err))
	}

	logger.Info("shutdown complete")
}
