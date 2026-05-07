package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewStore(db)
}

func TestCreateGroupWithCreatorOnlyUpdatesGroupID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	creator := &models.User{
		WeChatID:        "creator-wechat",
		Nickname:        "creator",
		Phone:           "13800138010",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, creator); err != nil {
		t.Fatalf("create creator: %v", err)
	}

	otherUser := &models.User{
		WeChatID:        "other-wechat",
		Nickname:        "other",
		Phone:           "13800138011",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, otherUser); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	creator.WeChatID = otherUser.WeChatID
	group := &models.Group{Name: "Warm Home", InviteCode: "WARM01", CreatorID: creator.ID}

	if err := store.CreateGroupWithCreator(ctx, creator, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	reloaded, err := store.GetUserByID(ctx, creator.ID)
	if err != nil {
		t.Fatalf("reload creator: %v", err)
	}
	if reloaded.GroupID == nil || *reloaded.GroupID != group.ID {
		t.Fatalf("expected creator group id %d, got %v", group.ID, reloaded.GroupID)
	}
	if reloaded.WeChatID != "creator-wechat" {
		t.Fatalf("expected creator wechat id to remain unchanged, got %s", reloaded.WeChatID)
	}
	if creator.GroupID == nil || *creator.GroupID != group.ID {
		t.Fatalf("expected in-memory creator group id %d, got %v", group.ID, creator.GroupID)
	}
}
