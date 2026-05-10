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

func TestRecordHandlerCreateNoteAcceptsLongContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-long-note-user", "record-handler-long-note-user", "13800138216")
	createRecordTestGroup(t, store, ctx, user, "record-handler-long-note-group", "NOTE16")

	content := strings.Repeat("长文本note", 400)
	body := bytes.NewBufferString(`{"content":` + strconv.Quote(content) + `,"color":"#fff","type":"normal"}`)
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
	if data["content"] != content {
		t.Fatalf("expected long content to round-trip, got %#v", data["content"])
	}
	if len(content) <= 1000 {
		t.Fatalf("expected test content to exceed 1000 chars, got %d", len(content))
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

func TestRecordHandlerGetNotesReturnsFolderTypeForViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	owner := createRecordTestUser(t, store, ctx, "record-handler-notes-owner", "record-handler-notes-owner", "13800138210")
	viewer := createRecordTestUser(t, store, ctx, "record-handler-notes-viewer", "record-handler-notes-viewer", "13800138211")
	group := createRecordTestGroup(t, store, ctx, owner, "record-handler-notes-group", "NOTE14")
	if err := store.AddUserToGroup(ctx, viewer, group.ID); err != nil {
		t.Fatalf("add viewer to group: %v", err)
	}
	folder := &models.NoteFolder{GroupID: group.ID, OwnerID: owner.ID, Name: "Trips"}
	if err := store.CreateNoteFolder(ctx, folder); err != nil {
		t.Fatalf("create note folder: %v", err)
	}
	note := &models.Note{GroupID: group.ID, AuthorID: owner.ID, FolderID: &folder.ID, Content: "shared note", Type: "normal", Visibility: "public"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/records/notes", nil)
	ginContext.Set(userIDContextKey, viewer.ID)

	handler.GetNotes(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items, ok := resp.Data.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one note item, got %#v", resp.Data)
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected note object, got %#v", items[0])
	}
	if item["folder_type"] != "shared" {
		t.Fatalf("expected folder_type shared, got %#v", item["folder_type"])
	}
	if _, exists := item["folder_id"]; exists {
		t.Fatalf("expected shared note to omit folder_id, got %#v", item["folder_id"])
	}
}

func TestRecordHandlerGetNotesFiltersByFolderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-folder-filter-user", "record-handler-folder-filter-user", "13800138212")
	group := createRecordTestGroup(t, store, ctx, user, "record-handler-folder-filter-group", "NOTE16")
	folderA := &models.NoteFolder{GroupID: group.ID, OwnerID: user.ID, Name: "A"}
	folderB := &models.NoteFolder{GroupID: group.ID, OwnerID: user.ID, Name: "B"}
	if err := store.CreateNoteFolder(ctx, folderA); err != nil {
		t.Fatalf("create note folder A: %v", err)
	}
	if err := store.CreateNoteFolder(ctx, folderB); err != nil {
		t.Fatalf("create note folder B: %v", err)
	}
	if err := store.CreateNote(ctx, &models.Note{GroupID: group.ID, AuthorID: user.ID, FolderID: &folderA.ID, Content: "note A", Type: "normal", Visibility: "public"}); err != nil {
		t.Fatalf("create note A: %v", err)
	}
	if err := store.CreateNote(ctx, &models.Note{GroupID: group.ID, AuthorID: user.ID, FolderID: &folderB.ID, Content: "note B", Type: "normal", Visibility: "public"}); err != nil {
		t.Fatalf("create note B: %v", err)
	}

	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/records/notes?folder_id="+strconv.FormatUint(uint64(folderA.ID), 10), nil)
	ginContext.Set(userIDContextKey, user.ID)

	handler.GetNotes(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items, ok := resp.Data.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one note item, got %#v", resp.Data)
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected note object, got %#v", items[0])
	}
	if item["content"] != "note A" {
		t.Fatalf("expected note A, got %#v", item["content"])
	}
	if item["folder_type"] != "custom" {
		t.Fatalf("expected folder_type custom, got %#v", item["folder_type"])
	}
}

func TestRecordHandlerMoveNoteToFolder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-move-note-user", "record-handler-move-note-user", "13800138213")
	group := createRecordTestGroup(t, store, ctx, user, "record-handler-move-note-group", "NOTE18")
	folder := &models.NoteFolder{GroupID: group.ID, OwnerID: user.ID, Name: "Inbox"}
	if err := store.CreateNoteFolder(ctx, folder); err != nil {
		t.Fatalf("create note folder: %v", err)
	}
	note := &models.Note{GroupID: group.ID, AuthorID: user.ID, Content: "move me", Type: "normal", Visibility: "private"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	body := bytes.NewBufferString(`{"folder_id":` + strconv.FormatUint(uint64(folder.ID), 10) + `}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/records/notes/1/folder", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(note.ID), 10)}}
	ginContext.Set(userIDContextKey, user.ID)

	handler.MoveNoteToFolder(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	reloaded, err := store.GetNoteByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("reload note: %v", err)
	}
	if reloaded.FolderID == nil || *reloaded.FolderID != folder.ID {
		t.Fatalf("expected folder id %d, got %+v", folder.ID, reloaded.FolderID)
	}
}

func TestRecordHandlerUpdateNoteVisibility(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-note-visibility-user", "record-handler-note-visibility-user", "13800138214")
	group := createRecordTestGroup(t, store, ctx, user, "record-handler-note-visibility-group", "NOTE19")
	note := &models.Note{GroupID: group.ID, AuthorID: user.ID, Content: "share me", Type: "normal", Visibility: "private"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	body := bytes.NewBufferString(`{"visibility":"public"}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPut, "/api/records/notes/1/visibility", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(note.ID), 10)}}
	ginContext.Set(userIDContextKey, user.ID)

	handler.UpdateNoteVisibility(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	reloaded, err := store.GetNoteByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("reload note: %v", err)
	}
	if reloaded.Visibility != "public" {
		t.Fatalf("expected public visibility, got %q", reloaded.Visibility)
	}
}

func TestRecordHandlerCreateAlbum(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, store := newTestRecordHandler(t)
	ctx := context.Background()
	user := createRecordTestUser(t, store, ctx, "record-handler-album-user", "record-handler-album-user", "13800138209")
	createRecordTestGroup(t, store, ctx, user, "record-handler-album-group", "ALBUM1")

	body := bytes.NewBufferString(`{"title":"trip","description":"memories","photos":[{"url":"https://example.com/1.jpg"}]}`)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/api/records/albums", body)
	ginContext.Request.Header.Set("Content-Type", "application/json")
	ginContext.Set(userIDContextKey, user.ID)

	handler.CreateAlbum(ginContext)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Message != "创建相册成功" {
		t.Fatalf("expected create album message, got %q", resp.Message)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data, got %#v", resp.Data)
	}
	likes, ok := data["likes"].(float64)
	if !ok {
		t.Fatalf("expected likes to be a number, got %#v", data["likes"])
	}
	if likes != 0 {
		t.Fatalf("expected likes count 0, got %v", likes)
	}
	photos, ok := data["photos"].([]any)
	if !ok || len(photos) != 1 {
		t.Fatalf("expected one photo in album response, got %#v", data["photos"])
	}
	photoData, ok := photos[0].(map[string]any)
	if !ok {
		t.Fatalf("expected photo object, got %#v", photos[0])
	}
	if _, exists := photoData["description"]; exists {
		t.Fatalf("expected photo description to be omitted, got %#v", photoData)
	}
	if _, exists := photoData["source"]; exists {
		t.Fatalf("expected photo source to be omitted, got %#v", photoData)
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
