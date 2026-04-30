package repository

import (
	"context"
	"time"
	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

// Store 封装所有数据库访问，避免 Handler 直接依赖 ORM 细节。
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByWeChatID(ctx context.Context, wechatID string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("wechat_id = ?", wechatID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByNickname(ctx context.Context, nickname string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("nickname = ?", nickname).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *Store) SaveUser(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Save(user).Error
}

func (s *Store) CreateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *Store) SaveGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *Store) GetGroupByInviteCode(ctx context.Context, inviteCode string) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).Where("invite_code = ?", inviteCode).First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) CreateGroupWithCreator(ctx context.Context, user *models.User, group *models.Group) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		user.GroupID = &group.ID
		return tx.Save(user).Error
	})
}

func (s *Store) AddUserToGroup(ctx context.Context, user *models.User, groupID uint) error {
	user.GroupID = &groupID
	return s.SaveUser(ctx, user)
}

func (s *Store) RemoveUserFromGroup(ctx context.Context, user *models.User) error {
	user.GroupID = nil
	return s.SaveUser(ctx, user)
}

func (s *Store) GetGroupWithMembers(ctx context.Context, groupID uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).Preload("Members").First(&group, groupID).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) UpdateGroupTimer(ctx context.Context, groupID uint, title string, startDate time.Time) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return nil, err
	}
	group.TimerTitle = title
	group.TimerStartDate = startDate
	if err := s.SaveGroup(ctx, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) UpdatePinnedMessage(ctx context.Context, groupID uint, authorID uint, content string) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return nil, err
	}
	group.PinnedMessage = content
	group.PinnedMessageAuthorID = authorID
	if err := s.SaveGroup(ctx, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) UpdateUserStatus(ctx context.Context, user *models.User, status string, updatedAt time.Time) error {
	user.CurrentStatus = status
	user.StatusUpdatedAt = updatedAt
	return s.SaveUser(ctx, user)
}

func (s *Store) GetLatestPhotoByGroup(ctx context.Context, groupID uint) (*models.Photo, error) {
	var photo models.Photo
	if err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Order("created_at desc").First(&photo).Error; err != nil {
		return nil, err
	}
	return &photo, nil
}

func (s *Store) GetLatestNoteByGroup(ctx context.Context, groupID uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Order("created_at desc").First(&note).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *Store) CreateAnniversary(ctx context.Context, anniversary *models.Anniversary) error {
	return s.db.WithContext(ctx).Create(anniversary).Error
}

func (s *Store) ListAnniversariesByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Anniversary, int64, error) {
	var anniversaries []models.Anniversary
	query := s.db.WithContext(ctx).Model(&models.Anniversary{}).Where("group_id = ?", groupID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("date asc").Offset(offset).Limit(limit).Find(&anniversaries).Error; err != nil {
		return nil, 0, err
	}
	return anniversaries, total, nil
}

func (s *Store) CreatePhoto(ctx context.Context, photo *models.Photo) error {
	return s.db.WithContext(ctx).Create(photo).Error
}

func (s *Store) ListPhotosByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	query := s.db.WithContext(ctx).Model(&models.Photo{}).Where("group_id = ?", groupID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Preload("Comments").Preload("Likes").Order("created_at desc").Offset(offset).Limit(limit).Find(&photos).Error; err != nil {
		return nil, 0, err
	}
	return photos, total, nil
}

func (s *Store) CreateNote(ctx context.Context, note *models.Note) error {
	return s.db.WithContext(ctx).Create(note).Error
}

func (s *Store) ListVisibleNotesByGroup(ctx context.Context, groupID uint, now time.Time, offset int, limit int) ([]models.Note, int64, error) {
	var notes []models.Note
	query := s.db.WithContext(ctx).Model(&models.Note{}).Where("group_id = ? AND (show_at IS NULL OR show_at <= ?)", groupID, now)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&notes).Error; err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func (s *Store) CreateWishlistItem(ctx context.Context, item *models.Wishlist) error {
	return s.db.WithContext(ctx).Create(item).Error
}

func (s *Store) ListWishlistByGroup(ctx context.Context, groupID uint, offset int, limit int) ([]models.Wishlist, int64, error) {
	var items []models.Wishlist
	query := s.db.WithContext(ctx).Model(&models.Wishlist{}).Where("group_id = ?", groupID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) GetWishlistItemByID(ctx context.Context, id uint) (*models.Wishlist, error) {
	var item models.Wishlist
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) SaveWishlistItem(ctx context.Context, item *models.Wishlist) error {
	return s.db.WithContext(ctx).Save(item).Error
}

func (s *Store) EnsureUserByWeChatID(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Where(models.User{WeChatID: user.WeChatID}).FirstOrCreate(user).Error
}

func (s *Store) EnsureGroupByInviteCode(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Where(models.Group{InviteCode: group.InviteCode}).FirstOrCreate(group).Error
}

func (s *Store) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	return s.db.WithContext(ctx).Create(token).Error
}

func (s *Store) GetRefreshTokenByTokenID(ctx context.Context, tokenID string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := s.db.WithContext(ctx).Where("token_id = ?", tokenID).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *Store) SaveRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	return s.db.WithContext(ctx).Save(token).Error
}
