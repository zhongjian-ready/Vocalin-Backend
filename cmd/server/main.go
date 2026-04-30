package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "vocalin-backend/docs" // Import generated docs
	"vocalin-backend/internal/app"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/routes"

	"go.uber.org/zap"
)

// @title Vocalin API
// @version 2.0
// @description Vocalin（窝聚）后端服务，基于 Gin + GORM + Viper + Zap + Validator + JWT + Swagger 构建
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 使用 Bearer <token> 传递 JWT 访问令牌
func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = application.Close() }()

	if err := database.AutoMigrate(application.DB); err != nil {
		application.Logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	r, err := routes.SetupRouter(application)
	if err != nil {
		application.Logger.Fatal("路由初始化失败", zap.Error(err))
	}

	server := &http.Server{
		Addr:         ":" + application.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  application.Config.Server.ReadTimeout,
		WriteTimeout: application.Config.Server.WriteTimeout,
	}

	go func() {
		application.Logger.Info("HTTP 服务启动", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			application.Logger.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		application.Logger.Fatal("HTTP 服务优雅关闭失败", zap.Error(err))
	}
	application.Logger.Info("HTTP 服务已停止")
}
