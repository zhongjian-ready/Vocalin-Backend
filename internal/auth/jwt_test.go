package auth

import (
	"testing"
	"time"

	"vocalin-backend/internal/config"
)

func TestTokenManagerGeneratesAndParsesDifferentTokenTypes(t *testing.T) {
	manager := NewTokenManager(config.AuthConfig{
		JWTSecret:       "test-secret",
		Issuer:          "test-issuer",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		ClockSkew:       time.Second,
	})

	accessToken, _, accessClaims, err := manager.GenerateAccessToken(7, "romeo")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if accessClaims.TokenType != TokenTypeAccess {
		t.Fatalf("expected access token type, got %s", accessClaims.TokenType)
	}
	if accessClaims.WeChatID != "romeo" {
		t.Fatalf("expected wechat id in claims, got %s", accessClaims.WeChatID)
	}

	refreshToken, _, refreshClaims, err := manager.GenerateRefreshToken(7, "romeo")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if refreshClaims.TokenType != TokenTypeRefresh {
		t.Fatalf("expected refresh token type, got %s", refreshClaims.TokenType)
	}

	if _, err := manager.ParseAccessToken(accessToken); err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if _, err := manager.ParseRefreshToken(refreshToken); err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if _, err := manager.ParseAccessToken(refreshToken); err == nil {
		t.Fatal("expected refresh token to be rejected by ParseAccessToken")
	}
	if _, err := manager.ParseRefreshToken(accessToken); err == nil {
		t.Fatal("expected access token to be rejected by ParseRefreshToken")
	}
}
