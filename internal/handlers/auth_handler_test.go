package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"vocalin-backend/internal/auth"
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/repository"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"
)

func TestAuthHandlerLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authService := newTestAuthHandler(t)
	ctx := context.Background()

	if _, err := authService.Register(ctx, "login-user", "13800138010", "secret123", "secret123"); err != nil {
		t.Fatalf("register user: %v", err)
	}

	body := bytes.NewBufferString(`{"nickname":"login-user","password":"secret123"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")

	handler.Login(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.Code)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", resp.Data)
	}
	if data["access_token"] == "" {
		t.Fatalf("expected access token in response, got %#v", data)
	}
	if data["refresh_token"] == "" {
		t.Fatalf("expected refresh token in response, got %#v", data)
	}
}

func TestAuthHandlerLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, authService := newTestAuthHandler(t)
	ctx := context.Background()

	loginResult, err := authService.Register(ctx, "logout-user", "13800138011", "secret123", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	body := bytes.NewBufferString(`{"refresh_token":"` + loginResult.RefreshToken + `"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, loginResult.User.ID)

	handler.Logout(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if _, err := authService.Refresh(ctx, loginResult.RefreshToken); err == nil {
		t.Fatal("expected logged-out refresh token to be revoked")
	}
}

func newTestAuthHandler(t *testing.T) (*AuthHandler, *service.AuthService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	store := repository.NewStore(db)
	tokenManager := auth.NewTokenManager(config.AuthConfig{
		JWTSecret:       "test-secret",
		Issuer:          "test-issuer",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		ClockSkew:       time.Second,
	})
	authService := service.NewAuthService(store, tokenManager, zap.NewNop())
	return NewAuthHandler(authService), authService
}
