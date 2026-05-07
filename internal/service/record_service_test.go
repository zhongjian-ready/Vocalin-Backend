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

	user := &models.User{WeChatID: "wechat-photo-user", Nickname: "photo-user", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "g1", InviteCode: "PHOTO1", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
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

	user := &models.User{WeChatID: "wishlist-user", Nickname: "wishlist-user", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.AddUserToGroup(ctx, user, group.ID); err != nil {
		t.Fatalf("add user to group: %v", err)
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

func TestRecordServiceCreateWishlistPersistsPriority(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "wishlist-priority-user", Nickname: "wishlist-priority-user", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "wishlist-priority-group", InviteCode: "WISH02", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	created, err := svc.CreateWishlist(ctx, user.ID, "plan a trip", "high")
	if err != nil {
		t.Fatalf("create wishlist: %v", err)
	}
	if created.Priority != "high" {
		t.Fatalf("expected created priority high, got %q", created.Priority)
	}

	result, err := svc.ListWishlist(ctx, user.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list wishlist: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 wishlist item, got %d", len(result.Items))
	}
	if result.Items[0].Priority != "high" {
		t.Fatalf("expected listed priority high, got %q", result.Items[0].Priority)
	}

	defaulted, err := svc.CreateWishlist(ctx, user.ID, "buy flowers", "")
	if err != nil {
		t.Fatalf("create wishlist with default priority: %v", err)
	}
	if defaulted.Priority != "medium" {
		t.Fatalf("expected default priority medium, got %q", defaulted.Priority)
	}
	stored, err := store.GetWishlistItemByID(ctx, defaulted.ID)
	if err != nil {
		t.Fatalf("reload wishlist item: %v", err)
	}
	if stored.Priority != "medium" {
		t.Fatalf("expected stored default priority medium, got %q", stored.Priority)
	}
}

func TestRecordServiceUpdateWishlistPriority(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "wishlist-update-user", Nickname: "wishlist-update-user", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "wishlist-update-group", InviteCode: "WISH03", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}
	item := &models.Wishlist{GroupID: group.ID, Content: "camping", Priority: "low"}
	if err := store.CreateWishlistItem(ctx, item); err != nil {
		t.Fatalf("create wishlist item: %v", err)
	}

	updated, err := svc.UpdateWishlistPriority(ctx, user.ID, item.ID, "high")
	if err != nil {
		t.Fatalf("update wishlist priority: %v", err)
	}
	if updated.Priority != "high" {
		t.Fatalf("expected updated priority high, got %q", updated.Priority)
	}

	reloaded, err := store.GetWishlistItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("reload wishlist item: %v", err)
	}
	if reloaded.Priority != "high" {
		t.Fatalf("expected persisted priority high, got %q", reloaded.Priority)
	}

	result, err := svc.ListWishlist(ctx, user.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list wishlist: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 wishlist item, got %d", len(result.Items))
	}
	if result.Items[0].Priority != "high" {
		t.Fatalf("expected listed priority high, got %q", result.Items[0].Priority)
	}
}
