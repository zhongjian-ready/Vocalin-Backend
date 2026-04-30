package service

import (
	"context"
	"testing"
	"time"

	"vocalin-backend/internal/models"
)

func TestRecordServiceListPhotosWithPagination(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	groupID := uint(1)
	user := &models.User{WeChatID: "wechat-photo-user", Nickname: "photo-user", GroupID: &groupID, StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "g1", InviteCode: "PHOTO1", CreatorID: user.ID}
	if err := store.CreateGroup(ctx, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	user.GroupID = &group.ID
	if err := store.SaveUser(ctx, user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	for index := 0; index < 3; index++ {
		photo := &models.Photo{GroupID: group.ID, UploaderID: user.ID, URL: "https://example.com/photo.jpg", Description: "photo"}
		if err := store.CreatePhoto(ctx, photo); err != nil {
			t.Fatalf("create photo: %v", err)
		}
	}

	result, err := svc.ListPhotos(ctx, user.ID, NewPagination(1, 2))
	if err != nil {
		t.Fatalf("list photos: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}
	if result.TotalPages != 2 {
		t.Fatalf("expected total pages 2, got %d", result.TotalPages)
	}
}

func TestRecordServiceIncompleteWishlist(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	group := &models.Group{Name: "wishlist-group", InviteCode: "WISH01"}
	if err := store.CreateGroup(ctx, group); err != nil {
		t.Fatalf("create group: %v", err)
	}

	user := &models.User{WeChatID: "wishlist-user", Nickname: "wishlist-user", GroupID: &group.ID, StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	completedAt := time.Now().Add(-time.Hour)
	item := &models.Wishlist{GroupID: group.ID, Content: "wish", IsCompleted: true, CompletedAt: &completedAt}
	if err := store.CreateWishlistItem(ctx, item); err != nil {
		t.Fatalf("create wishlist item: %v", err)
	}

	updated, err := svc.IncompleteWishlist(ctx, user.ID, item.ID)
	if err != nil {
		t.Fatalf("incomplete wishlist: %v", err)
	}
	if updated.IsCompleted {
		t.Fatal("expected wishlist item to be marked incomplete")
	}
	if updated.CompletedAt != nil {
		t.Fatalf("expected completed_at to be cleared, got %v", updated.CompletedAt)
	}

	reloaded, err := store.GetWishlistItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("reload wishlist item: %v", err)
	}
	if reloaded.IsCompleted {
		t.Fatal("expected persisted wishlist item to be incomplete")
	}
	if reloaded.CompletedAt != nil {
		t.Fatalf("expected persisted completed_at to be nil, got %v", reloaded.CompletedAt)
	}
}
