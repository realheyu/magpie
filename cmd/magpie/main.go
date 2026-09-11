package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/realheyu/magpie/internal/config"
	"github.com/realheyu/magpie/internal/logging"
	"github.com/realheyu/magpie/internal/security"
	"github.com/realheyu/magpie/internal/server"
	"github.com/realheyu/magpie/internal/store"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, err := logging.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()
	zap.ReplaceGlobals(logger)
	gin.SetMode(cfg.Server.GinMode)

	db, err := store.OpenWithLogger(cfg.MySQLDSN(), logging.NewGormLogger(logger, cfg.Log.Level))
	if err != nil {
		logger.Fatal("打开数据库失败", zap.Error(err))
	}
	defer db.Close()

	if err := db.AutoMigrate(); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}
	if err := db.BootstrapAdmin(cfg.Bootstrap.Username, cfg.Bootstrap.Password); err != nil {
		logger.Fatal("初始化管理员失败", zap.Error(err))
	}

	sessions := security.NewSessionManager(cfg.SessionTTL())
	adminServer := &http.Server{Addr: cfg.Server.AdminAddr, Handler: server.AdminRouter(db, sessions, cfg.Log.GinRequest)}
	apiServer := &http.Server{Addr: cfg.Server.APIAddr, Handler: server.ConfigAPIRouter(db, cfg.Log.GinRequest)}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go serve("admin", adminServer, errCh)
	go serve("api", apiServer, errCh)

	select {
	case <-ctx.Done():
		logger.Info("收到关闭信号")
	case err := <-errCh:
		logger.Error("服务异常停止", zap.Error(err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = adminServer.Shutdown(shutdownCtx)
	_ = apiServer.Shutdown(shutdownCtx)
}

func serve(name string, srv *http.Server, errCh chan<- error) {
	zap.L().Info("服务启动", zap.String("name", name), zap.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- err
	}
}
