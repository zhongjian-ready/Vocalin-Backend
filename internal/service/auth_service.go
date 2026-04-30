package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vocalin-backend/internal/auth"
	"vocalin-backend/internal/models"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	baseService
	tokenManager TokenManager
}

type LoginResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  *models.User
}

type RefreshResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type TokenManager interface {
	GenerateAccessToken(userID uint, weChatID string) (string, time.Time, *auth.Claims, error)
	GenerateRefreshToken(userID uint, weChatID string) (string, time.Time, *auth.Claims, error)
	ParseRefreshToken(tokenString string) (*auth.Claims, error)
}

func NewAuthService(store Store, tokenManager TokenManager, logger *zap.Logger) *AuthService {
	return &AuthService{
		baseService:  newBaseService(store, logger.Named("auth-service")),
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Register(ctx context.Context, nickname, phone, password, confirmPassword string) (*LoginResult, error) {
	nickname = strings.TrimSpace(nickname)
	phone = strings.TrimSpace(phone)
	if password != confirmPassword {
		return nil, ErrPasswordMismatch
	}
	if _, err := s.store.GetUserByNickname(ctx, nickname); err == nil {
		return nil, ErrNicknameAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("query nickname: %w", err)
	}
	if _, err := s.store.GetUserByPhone(ctx, phone); err == nil {
		return nil, ErrPhoneAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("query phone: %w", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &models.User{
		WeChatID:        fmt.Sprintf("phone:%s", phone),
		Nickname:        nickname,
		Phone:           phone,
		PasswordHash:    string(passwordHash),
		StatusUpdatedAt: time.Now(),
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	s.logger.Info("用户注册成功", zap.Uint("user_id", user.ID), zap.String("phone", user.Phone))
	return s.issueTokens(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, nickname, password string) (*LoginResult, error) {
	user, err := s.store.GetUserByNickname(ctx, strings.TrimSpace(nickname))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	s.logger.Info("用户登录成功", zap.Uint("user_id", user.ID), zap.String("nickname", user.Nickname))
	return s.issueTokens(ctx, user)
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User) (*LoginResult, error) {
	accessToken, accessExpiresAt, _, err := s.tokenManager.GenerateAccessToken(user.ID, user.WeChatID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, refreshExpiresAt, refreshClaims, err := s.tokenManager.GenerateRefreshToken(user.ID, user.WeChatID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	if err := s.store.CreateRefreshToken(ctx, &models.RefreshToken{
		TokenID:   refreshClaims.TokenID,
		UserID:    user.ID,
		ExpiresAt: refreshExpiresAt,
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt,
		User:                  user,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	storedToken, err := s.store.GetRefreshTokenByTokenID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("load refresh token: %w", err)
	}
	if storedToken.RevokedAt != nil || time.Now().After(storedToken.ExpiresAt) {
		return nil, ErrRefreshTokenRevoked
	}
	user, err := s.store.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	newAccessToken, newAccessExpiresAt, _, err := s.tokenManager.GenerateAccessToken(user.ID, user.WeChatID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	newRefreshToken, newRefreshExpiresAt, newRefreshClaims, err := s.tokenManager.GenerateRefreshToken(user.ID, user.WeChatID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	now := time.Now()
	storedToken.RevokedAt = &now
	storedToken.ReplacedByTokenID = newRefreshClaims.TokenID
	if err := s.store.SaveRefreshToken(ctx, storedToken); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}
	if err := s.store.CreateRefreshToken(ctx, &models.RefreshToken{
		TokenID:   newRefreshClaims.TokenID,
		UserID:    user.ID,
		ExpiresAt: newRefreshExpiresAt,
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}
	return &RefreshResult{
		AccessToken:           newAccessToken,
		AccessTokenExpiresAt:  newAccessExpiresAt,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshExpiresAt,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uint, refreshToken string) error {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}
	if claims.UserID != userID {
		return ErrForbidden
	}
	storedToken, err := s.store.GetRefreshTokenByTokenID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidRefreshToken
		}
		return fmt.Errorf("load refresh token: %w", err)
	}
	if storedToken.RevokedAt != nil {
		return ErrRefreshTokenRevoked
	}
	now := time.Now()
	storedToken.RevokedAt = &now
	if err := s.store.SaveRefreshToken(ctx, storedToken); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
