package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"vocalin-backend/internal/config"
)

var ErrInvalidToken = errors.New("invalid token")

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims 保存当前请求需要的用户身份信息。
type Claims struct {
	UserID    uint   `json:"user_id"`
	WeChatID  string `json:"wechat_id"`
	TokenType string `json:"token_type"`
	TokenID   string `json:"token_id"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret          []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	clockSkew       time.Duration
}

func NewTokenManager(cfg config.AuthConfig) *TokenManager {
	return &TokenManager{
		secret:          []byte(cfg.JWTSecret),
		issuer:          cfg.Issuer,
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		clockSkew:       cfg.ClockSkew,
	}
}

func (m *TokenManager) GenerateAccessToken(userID uint, weChatID string) (string, time.Time, *Claims, error) {
	return m.generate(userID, weChatID, TokenTypeAccess, m.accessTokenTTL)
}

func (m *TokenManager) GenerateRefreshToken(userID uint, weChatID string) (string, time.Time, *Claims, error) {
	return m.generate(userID, weChatID, TokenTypeRefresh, m.refreshTokenTTL)
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*Claims, error) {
	return m.parse(tokenString, TokenTypeAccess)
}

func (m *TokenManager) ParseRefreshToken(tokenString string) (*Claims, error) {
	return m.parse(tokenString, TokenTypeRefresh)
}

func (m *TokenManager) generate(userID uint, weChatID, tokenType string, ttl time.Duration) (string, time.Time, *Claims, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)
	tokenID, err := generateTokenID()
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("generate token id: %w", err)
	}

	claims := Claims{
		UserID:    userID,
		WeChatID:  weChatID,
		TokenType: tokenType,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-m.clockSkew)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, &claims, nil
}

func (m *TokenManager) parse(tokenString string, expectedType string) (*Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithLeeway(m.clockSkew))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func generateTokenID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
