package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"vocalin-backend/internal/clock"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/repository"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"
)

func TestProfileHandlerUpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, store := newTestProfileHandler(t)
	ctx := context.Background()

	user := createProfileTestUser(t, store, ctx, "profile-handler-user", "profile-handler-user", "13800138201")

	body := bytes.NewBufferString(`{"avatar_url":"https://example.com/avatar.png","nickname":"John","status":"Running on snacks"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/profile/update", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.UpdateProfile(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", resp.Data)
	}
	if data["nickname"] != "John" {
		t.Fatalf("expected nickname John, got %#v", data["nickname"])
	}
	if data["avatar_url"] != "https://example.com/avatar.png" {
		t.Fatalf("expected avatar_url to be updated, got %#v", data["avatar_url"])
	}
	if data["current_status"] != "Running on snacks" {
		t.Fatalf("expected current_status to be updated, got %#v", data["current_status"])
	}
}

func TestProfileHandlerUpdateProfileRejectsDuplicateNickname(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, store := newTestProfileHandler(t)
	ctx := context.Background()

	user := createProfileTestUser(t, store, ctx, "profile-handler-owner", "owner-name", "13800138202")
	createProfileTestUser(t, store, ctx, "profile-handler-taken", "taken-name", "13800138203")

	body := bytes.NewBufferString(`{"nickname":"taken-name","status":"Still here"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/profile/update", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.UpdateProfile(ginContext)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != "AUTH_REGISTER_CONFLICT" {
		t.Fatalf("expected AUTH_REGISTER_CONFLICT, got %s", resp.Code)
	}
}

func TestProfileHandlerUpdateProfileRejectsBlankNickname(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, store := newTestProfileHandler(t)
	ctx := context.Background()

	user := createProfileTestUser(t, store, ctx, "profile-handler-blank", "blank-name", "13800138204")

	body := bytes.NewBufferString(`{"nickname":"   ","status":"Still here"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/profile/update", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.UpdateProfile(ginContext)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestProfileHandlerUpdateProfileUsesChinaTimezoneInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := time.Local
	t.Cleanup(func() {
		time.Local = original
	})

	if err := clock.SetSystemLocationToChina(); err != nil {
		t.Fatalf("set timezone: %v", err)
	}

	handler, store := newTestProfileHandler(t)
	ctx := context.Background()

	user := createProfileTestUser(t, store, ctx, "profile-handler-timezone", "timezone-name", "13800138205")

	body := bytes.NewBufferString(`{"avatar_url":"https://example.com/avatar.png","nickname":"Timezone John","status":"Running on snacks"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/profile/update", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.UpdateProfile(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", resp.Data)
	}

	statusUpdatedAt, ok := data["status_updated_at"].(string)
	if !ok {
		t.Fatalf("expected status_updated_at string, got %#v", data["status_updated_at"])
	}
	if !strings.HasSuffix(statusUpdatedAt, "+08:00") {
		t.Fatalf("expected status_updated_at to use +08:00, got %q", statusUpdatedAt)
	}
	if strings.HasSuffix(statusUpdatedAt, "Z") {
		t.Fatalf("expected status_updated_at not to use UTC suffix, got %q", statusUpdatedAt)
	}
}

func newTestProfileHandler(t *testing.T) (*ProfileHandler, *repository.Store) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	store := repository.NewStore(db)
	profileService := service.NewProfileService(store, zap.NewNop())
	return NewProfileHandler(profileService), store
}

func createProfileTestUser(t *testing.T, store *repository.Store, ctx context.Context, wechatID string, nickname string, phone string) *models.User {
	t.Helper()
	user := &models.User{
		WeChatID:        wechatID,
		Nickname:        nickname,
		Phone:           phone,
		PasswordHash:    "hashed",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}
