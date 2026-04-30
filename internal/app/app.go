package app

import (
	"fmt"
	"vocalin-backend/internal/auth"
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/logger"
	"vocalin-backend/internal/repository"
	"vocalin-backend/internal/service"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App 聚合应用启动时需要的所有基础依赖。
type App struct {
	Config       *config.Config
	Logger       *zap.Logger
	DB           *gorm.DB
	Store        *repository.Store
	TokenManager *auth.TokenManager
	Services     *service.Services
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	db, err := database.New(cfg.Database, log)
	if err != nil {
		return nil, err
	}

	store := repository.NewStore(db)
	tokenManager := auth.NewTokenManager(cfg.Auth)
	services := service.NewServices(store, tokenManager, log)

	return &App{
		Config:       cfg,
		Logger:       log,
		DB:           db,
		Store:        store,
		TokenManager: tokenManager,
		Services:     services,
	}, nil
}

func (a *App) Close() error {
	if a.Logger != nil {
		_ = a.Logger.Sync()
	}
	if a.DB == nil {
		return nil
	}
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
