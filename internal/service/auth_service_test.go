package service

import (
	"context"
	"testing"
)

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	store := newTestStore(t)
	service := NewAuthService(store, newTestTokenManager(), newTestLogger())
	ctx := context.Background()

	registerResult, err := service.Register(ctx, "tester", "13800138000", "secret123", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if registerResult.User.Phone != "13800138000" {
		t.Fatalf("expected phone to be persisted, got %s", registerResult.User.Phone)
	}
	if registerResult.User.PasswordHash == "" {
		t.Fatal("expected password hash to be stored")
	}

	loginResult, err := service.Login(ctx, "tester", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginResult.User.ID != registerResult.User.ID {
		t.Fatalf("expected login to return registered user, got %d and %d", loginResult.User.ID, registerResult.User.ID)
	}
	if _, err := service.Login(ctx, "tester", "bad-password"); err == nil {
		t.Fatal("expected invalid password to be rejected")
	}
	if _, err := service.Register(ctx, "tester", "13900139000", "secret123", "secret123"); err == nil {
		t.Fatal("expected duplicate nickname to be rejected")
	}
}

func TestAuthServiceRefreshAndLogout(t *testing.T) {
	store := newTestStore(t)
	service := NewAuthService(store, newTestTokenManager(), newTestLogger())
	ctx := context.Background()

	loginResult, err := service.Register(ctx, "tester-refresh", "13800138001", "secret123", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if loginResult.RefreshToken == "" || loginResult.AccessToken == "" {
		t.Fatal("expected both access token and refresh token")
	}
	if loginResult.User.Phone != "13800138001" {
		t.Fatalf("expected user phone to be persisted, got %s", loginResult.User.Phone)
	}

	refreshed, err := service.Refresh(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.RefreshToken == loginResult.RefreshToken {
		t.Fatal("expected refresh rotation to issue a new refresh token")
	}

	if err := service.Logout(ctx, loginResult.User.ID, refreshed.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := service.Refresh(ctx, refreshed.RefreshToken); err == nil {
		t.Fatal("expected revoked refresh token to be rejected")
	}
}

func TestAuthServiceRegisterRejectsPasswordMismatch(t *testing.T) {
	store := newTestStore(t)
	service := NewAuthService(store, newTestTokenManager(), newTestLogger())
	ctx := context.Background()

	if _, err := service.Register(ctx, "tester", "13800138009", "secret123", "secret456"); err == nil {
		t.Fatal("expected password mismatch to be rejected")
	}
}
