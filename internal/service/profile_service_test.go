package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"vocalin-backend/internal/models"
)

func TestProfileServiceUpdateProfile(t *testing.T) {
	store := newTestStore(t)
	svc := NewProfileService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{
		WeChatID:        "wechat-profile-success",
		Nickname:        "old-name",
		Phone:           "13800138101",
		PasswordHash:    "hashed",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	previousUpdatedAt := user.StatusUpdatedAt

	updated, err := svc.UpdateProfile(ctx, user.ID, "  John  ", "https://example.com/avatar.png", "  Running on snacks  ")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Nickname != "John" {
		t.Fatalf("expected trimmed nickname, got %q", updated.Nickname)
	}
	if updated.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("expected avatar url to be updated, got %q", updated.AvatarURL)
	}
	if updated.CurrentStatus != "Running on snacks" {
		t.Fatalf("expected trimmed status, got %q", updated.CurrentStatus)
	}
	if !updated.StatusUpdatedAt.After(previousUpdatedAt) {
		t.Fatal("expected status update time to move forward")
	}

	persisted, err := store.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if persisted.Nickname != "John" || persisted.AvatarURL != "https://example.com/avatar.png" || persisted.CurrentStatus != "Running on snacks" {
		t.Fatalf("unexpected persisted user: %#v", persisted)
	}
}

func TestProfileServiceUpdateProfileRejectsEmptyNickname(t *testing.T) {
	store := newTestStore(t)
	svc := NewProfileService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{
		WeChatID:        "wechat-profile-empty",
		Nickname:        "existing-name",
		Phone:           "13800138102",
		PasswordHash:    "hashed",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err := svc.UpdateProfile(ctx, user.ID, "   ", "", "")
	if !errors.Is(err, ErrNicknameRequired) {
		t.Fatalf("expected ErrNicknameRequired, got %v", err)
	}
}

func TestProfileServiceUpdateProfileRejectsDuplicateNickname(t *testing.T) {
	store := newTestStore(t)
	svc := NewProfileService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{
		WeChatID:        "wechat-profile-owner",
		Nickname:        "owner-name",
		Phone:           "13800138103",
		PasswordHash:    "hashed",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	other := &models.User{
		WeChatID:        "wechat-profile-other",
		Nickname:        "taken-name",
		Phone:           "13800138104",
		PasswordHash:    "hashed",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, other); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	_, err := svc.UpdateProfile(ctx, owner.ID, "taken-name", "", "")
	if !errors.Is(err, ErrNicknameAlreadyExists) {
		t.Fatalf("expected ErrNicknameAlreadyExists, got %v", err)
	}
	if owner.Nickname != "owner-name" {
		t.Fatalf("expected nickname to remain unchanged, got %q", owner.Nickname)
	}
}
