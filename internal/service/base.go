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
	GetGroupByID(ctx context.Context, groupID uint) (*models.Group, error)
	GetGroupMember(ctx context.Context, groupID uint, userID uint) (*models.GroupMember, error)
	ListGroupMembersByUser(ctx context.Context, userID uint) ([]models.GroupMember, error)
	SetCurrentGroup(ctx context.Context, userID uint, groupID *uint) error
	ListGroupsByUser(ctx context.Context, userID uint) ([]models.Group, error)
	CountGroupMembers(ctx context.Context, groupID uint) (int64, error)
	GetFirstGroupByUser(ctx context.Context, userID uint) (*models.Group, error)
	GetGroupByInviteCode(ctx context.Context, inviteCode string) (*models.Group, error)
	CreateGroupWithCreator(ctx context.Context, user *models.User, group *models.Group) error
	AddUserToGroup(ctx context.Context, user *models.User, groupID uint) error
	RemoveUserFromGroup(ctx context.Context, userID uint, groupID uint) (*uint, error)
	TransferGroupOwnership(ctx context.Context, groupID uint, targetUserID uint) error
	CreateGroupRequest(ctx context.Context, request *models.GroupRequest) error
	GetGroupRequestByID(ctx context.Context, requestID uint) (*models.GroupRequest, error)
	FindPendingJoinRequest(ctx context.Context, groupID uint, requesterUserID uint) (*models.GroupRequest, error)
	FindPendingOwnershipTransferRequest(ctx context.Context, groupID uint) (*models.GroupRequest, error)
	ListPendingGroupRequestsByRequester(ctx context.Context, requesterUserID uint) ([]models.GroupRequest, error)
	ListPendingGroupRequestsForTarget(ctx context.Context, targetUserID uint) ([]models.GroupRequest, error)
	CountPendingGroupRequestsForTarget(ctx context.Context, targetUserID uint) (int64, error)
	ApproveGroupRequest(ctx context.Context, requestID uint, reviewerUserID uint) error
	RejectGroupRequest(ctx context.Context, requestID uint, reviewerUserID uint) error
	DisbandGroup(ctx context.Context, groupID uint) (map[uint]*uint, error)
	GetGroupWithMembers(ctx context.Context, groupID uint) (*models.Group, error)
	UpdatePinnedMessage(ctx context.Context, groupID uint, authorID uint, content string) (*models.Group, error)
	UpdateUserStatus(ctx context.Context, user *models.User, status string, updatedAt time.Time) error
	GetLatestVisibleAlbumByGroup(ctx context.Context, groupID uint, viewerID uint) (*models.Album, error)
	GetLatestVisibleNoteByGroup(ctx context.Context, groupID uint, viewerID uint, now time.Time) (*models.Note, error)
	CreateAnniversary(ctx context.Context, anniversary *models.Anniversary) error
	ListAnniversariesByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Anniversary, int64, error)
	CreateAlbum(ctx context.Context, album *models.Album) error
	GetAlbumByID(ctx context.Context, id uint) (*models.Album, error)
	SaveAlbum(ctx context.Context, album *models.Album) error
	ReplaceAlbumPhotos(ctx context.Context, albumID uint, photos []models.Photo) error
	DeleteAlbum(ctx context.Context, id uint) error
	ListAlbumsByGroup(ctx context.Context, groupID uint, viewerID uint, offset int, limit int) ([]models.Album, int64, error)
	CreateNoteFolder(ctx context.Context, folder *models.NoteFolder) error
	GetNoteFolderByID(ctx context.Context, id uint) (*models.NoteFolder, error)
	SaveNoteFolder(ctx context.Context, folder *models.NoteFolder) error
	DeleteNoteFolder(ctx context.Context, id uint, ownerID uint) error
	ListNoteFoldersByOwner(ctx context.Context, groupID uint, ownerID uint) ([]models.NoteFolder, error)
	CreateNote(ctx context.Context, note *models.Note) error
	GetNoteByID(ctx context.Context, id uint) (*models.Note, error)
	SaveNote(ctx context.Context, note *models.Note) error
	DeleteNote(ctx context.Context, id uint) error
	ListVisibleNotesByGroup(ctx context.Context, groupID uint, viewerID uint, now time.Time, offset int, limit int, folderType string, folderID *uint) ([]models.Note, int64, error)
	CreateWishlistItem(ctx context.Context, item *models.Wishlist) error
	ListWishlistByGroup(ctx context.Context, groupID uint, viewerID uint, offset int, limit int) ([]models.Wishlist, int64, error)
	GetWishlistItemByID(ctx context.Context, id uint) (*models.Wishlist, error)
	SaveWishlistItem(ctx context.Context, item *models.Wishlist) error
	DeleteWishlistItem(ctx context.Context, id uint) error
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
	if user.CurrentGroupID == nil {
		return nil, 0, ErrUserNotInGroup
	}
	if _, err := s.store.GetGroupMember(ctx, *user.CurrentGroupID, user.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrUserNotInGroup
		}
		return nil, 0, err
	}
	return user, *user.CurrentGroupID, nil
}

func applyGroupMembershipMetadata(group *models.Group, membership *models.GroupMember) {
	group.MyRole = membership.Role
	joinedAt := membership.CreatedAt
	group.TimerStartDate = &joinedAt
}
