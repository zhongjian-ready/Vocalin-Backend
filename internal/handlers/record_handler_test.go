package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	appvalidator "vocalin-backend/internal/validator"
)

func TestRecordHandlerCreateNoteUsesChinaTimezoneInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := time.Local
	t.Cleanup(func() {
		time.Local = original
	})

	if err := clock.SetSystemLocationToChina(); err != nil {
		t.Fatalf("set timezone: %v", err)
	}

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-user", "record-handler-user", "13800138206")
	createRecordTestGroup(t, store, ctx, user, "record-handler-group", "NOTE08")

	body := bytes.NewBufferString(`{"content":"hello","color":"#fff","type":"normal"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/records/notes", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.CreateNote(ginContext)

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

	createdAt, ok := data["CreatedAt"].(string)
	if !ok {
		t.Fatalf("expected CreatedAt string, got %#v", data["CreatedAt"])
	}
	if !strings.HasSuffix(createdAt, "+08:00") {
		t.Fatalf("expected CreatedAt to use +08:00, got %q", createdAt)
	}
	if strings.HasSuffix(createdAt, "Z") {
		t.Fatalf("expected CreatedAt not to use UTC suffix, got %q", createdAt)
	}
}

func TestRecordHandlerCreateTimedNoteNormalizesShowAtToChinaTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := time.Local
	t.Cleanup(func() {
		time.Local = original
	})

	if err := clock.SetSystemLocationToChina(); err != nil {
		t.Fatalf("set timezone: %v", err)
	}

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-timed-user", "record-handler-timed-user", "13800138207")
	createRecordTestGroup(t, store, ctx, user, "record-handler-timed-group", "NOTE09")

	body := bytes.NewBufferString(`{"content":"hello","color":"#fff","type":"timed","show_at":"2026-05-08T06:00:00Z"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/records/notes", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.CreateNote(ginContext)

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

	showAt, ok := data["show_at"].(string)
	if !ok {
		t.Fatalf("expected show_at string, got %#v", data["show_at"])
	}
	if !strings.HasSuffix(showAt, "+08:00") {
		t.Fatalf("expected show_at to use +08:00, got %q", showAt)
	}
	if strings.HasSuffix(showAt, "Z") {
		t.Fatalf("expected show_at not to use UTC suffix, got %q", showAt)
	}
}

func TestRecordHandlerDeleteNote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-delete-user", "record-handler-delete-user", "13800138208")
	group := createRecordTestGroup(t, store, ctx, user, "record-handler-delete-group", "NOTE12")
	note := &models.Note{GroupID: group.ID, AuthorID: user.ID, Content: "delete me", Type: "normal"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodDelete, "/api/records/notes/1", nil)
	ginContext.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(note.ID), 10)}}
	ginContext.Set(userIDContextKey, user.ID)

	handler.DeleteNote(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != "SUCCESS" {
		t.Fatalf("expected success code, got %q", resp.Code)
	}
	if resp.Message != "删除便签成功" {
		t.Fatalf("expected delete message, got %q", resp.Message)
	}

	if _, err := store.GetNoteByID(ctx, note.ID); err == nil {
		t.Fatal("expected note to be deleted")
	}
}

func newTestRecordHandler(t *testing.T) (*RecordHandler, *repository.Store) {
	t.Helper()
	if err := appvalidator.Register(); err != nil {
		t.Fatalf("register validator: %v", err)
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	store := repository.NewStore(db)
	recordService := service.NewRecordService(store, zap.NewNop())
	return NewRecordHandler(recordService), store
}

func createRecordTestUser(t *testing.T, store *repository.Store, ctx context.Context, wechatID string, nickname string, phone string) *models.User {
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

func createRecordTestGroup(t *testing.T, store *repository.Store, ctx context.Context, user *models.User, name string, inviteCode string) *models.Group {
	t.Helper()
	group := &models.Group{Name: name, InviteCode: inviteCode, CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create test group: %v", err)
	}
	return group
}
