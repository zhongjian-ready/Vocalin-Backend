package service

import (
	"testing"
	"time"
	"vocalin-backend/internal/auth"
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/repository"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestStore(t *testing.T) *repository.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return repository.NewStore(db)
}

func newTestTokenManager() *auth.TokenManager {
	return auth.NewTokenManager(config.AuthConfig{
		JWTSecret:       "test-secret",
		Issuer:          "test-issuer",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		ClockSkew:       time.Second,
	})
}

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}
