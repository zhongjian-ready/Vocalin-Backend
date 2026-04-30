package service

import (
	"context"
	"errors"
	"time"

	"vocalin-backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Store interface {
	GetUserByID(ctx context.Context, userID uint) (*models.User, error)
	GetUserByWeChatID(ctx context.Context, wechatID string) (*models.User, error)
	GetUserByNickname(ctx context.Context, nickname string) (*models.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	SaveUser(ctx context.Context, user *models.User) error
	CreateGroup(ctx context.Context, group *models.Group) error
	SaveGroup(ctx context.Context, group *models.Group) error
	GetGroupByInviteCode(ctx context.Context, inviteCode string) (*models.Group, error)
	CreateGroupWithCreator(ctx context.Context, user *models.User, group *models.Group) error
	AddUserToGroup(ctx context.Context, user *models.User, groupID uint) error
	RemoveUserFromGroup(ctx context.Context, user *models.User) error
	GetGroupWithMembers(ctx context.Context, groupID uint) (*models.Group, error)
	UpdateGroupTimer(ctx context.Context, groupID uint, title string, startDate time.Time) (*models.Group, error)
	UpdatePinnedMessage(ctx context.Context, groupID uint, authorID uint, content string) (*models.Group, error)
	UpdateUserStatus(ctx context.Context, user *models.User, status string, updatedAt time.Time) error
	GetLatestPhotoByGroup(ctx context.Context, groupID uint) (*models.Photo, error)
	GetLatestNoteByGroup(ctx context.Context, groupID uint) (*models.Note, error)
	CreateAnniversary(ctx context.Context, anniversary *models.Anniversary) error
	ListAnniversariesByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Anniversary, int64, error)
	CreatePhoto(ctx context.Context, photo *models.Photo) error
	ListPhotosByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Photo, int64, error)
	CreateNote(ctx context.Context, note *models.Note) error
	ListVisibleNotesByGroup(ctx context.Context, groupID uint, now time.Time, offset int, limit int) ([]models.Note, int64, error)
	CreateWishlistItem(ctx context.Context, item *models.Wishlist) error
	ListWishlistByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Wishlist, int64, error)
	GetWishlistItemByID(ctx context.Context, id uint) (*models.Wishlist, error)
	SaveWishlistItem(ctx context.Context, item *models.Wishlist) error
	EnsureUserByWeChatID(ctx context.Context, user *models.User) error
	EnsureGroupByInviteCode(ctx context.Context, group *models.Group) error
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshTokenByTokenID(ctx context.Context, tokenID string) (*models.RefreshToken, error)
	SaveRefreshToken(ctx context.Context, token *models.RefreshToken) error
}

type baseService struct {
	store  Store
	logger *zap.Logger
}

func newBaseService(store Store, logger *zap.Logger) baseService {
	return baseService{store: store, logger: logger}
}

func (s *baseService) currentUser(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	return user, nil
}

func (s *baseService) currentGroupUser(ctx context.Context, userID uint) (*models.User, uint, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if user.GroupID == nil {
		return nil, 0, ErrUserNotInGroup
	}
	return user, *user.GroupID, nil
}
