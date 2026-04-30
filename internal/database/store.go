package database

import (
	"time"
	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Store) GetUserByWeChatID(wechatID string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("wechat_id = ?", wechatID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Store) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *Store) SaveUser(user *models.User) error {
	return s.db.Save(user).Error
}

func (s *Store) CreateGroup(group *models.Group) error {
	return s.db.Create(group).Error
}

func (s *Store) SaveGroup(group *models.Group) error {
	return s.db.Save(group).Error
}

func (s *Store) GetGroupByInviteCode(inviteCode string) (*models.Group, error) {
	var group models.Group
	if err := s.db.Where("invite_code = ?", inviteCode).First(&group).Error; err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *Store) CreateGroupWithCreator(user *models.User, group *models.Group) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		user.GroupID = &group.ID
		return tx.Save(user).Error
	})
}

func (s *Store) AddUserToGroup(user *models.User, groupID uint) error {
	user.GroupID = &groupID
	return s.SaveUser(user)
}

func (s *Store) RemoveUserFromGroup(user *models.User) error {
	user.GroupID = nil
	return s.SaveUser(user)
}

func (s *Store) GetGroupWithMembers(groupID uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.Preload("Members").First(&group, groupID).Error; err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *Store) UpdateGroupTimer(groupID uint, title string, startDate time.Time) (*models.Group, error) {
	var group models.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return nil, err
	}

	group.TimerTitle = title
	group.TimerStartDate = startDate
	if err := s.SaveGroup(&group); err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *Store) UpdatePinnedMessage(groupID uint, authorID uint, content string) (*models.Group, error) {
	var group models.Group
	if err := s.db.First(&group, groupID).Error; err != nil {
		return nil, err
	}

	group.PinnedMessage = content
	group.PinnedMessageAuthorID = authorID
	if err := s.SaveGroup(&group); err != nil {
		return nil, err
	}

	return &group, nil
}

func (s *Store) UpdateUserStatus(user *models.User, status string, updatedAt time.Time) error {
	user.CurrentStatus = status
	user.StatusUpdatedAt = updatedAt
	return s.SaveUser(user)
}

func (s *Store) GetLatestPhotoByGroup(groupID uint) (*models.Photo, error) {
	var photo models.Photo
	if err := s.db.Where("group_id = ?", groupID).Order("created_at desc").First(&photo).Error; err != nil {
		return nil, err
	}

	return &photo, nil
}

func (s *Store) GetLatestNoteByGroup(groupID uint) (*models.Note, error) {
	var note models.Note
	if err := s.db.Where("group_id = ?", groupID).Order("created_at desc").First(&note).Error; err != nil {
		return nil, err
	}

	return &note, nil
}

func (s *Store) CreateAnniversary(anniversary *models.Anniversary) error {
	return s.db.Create(anniversary).Error
}

func (s *Store) ListAnniversariesByGroup(groupID uint) ([]models.Anniversary, error) {
	var anniversaries []models.Anniversary
	if err := s.db.Where("group_id = ?", groupID).Order("date asc").Find(&anniversaries).Error; err != nil {
		return nil, err
	}

	return anniversaries, nil
}

func (s *Store) CreatePhoto(photo *models.Photo) error {
	return s.db.Create(photo).Error
}

func (s *Store) ListPhotosByGroup(groupID uint) ([]models.Photo, error) {
	var photos []models.Photo
	if err := s.db.Where("group_id = ?", groupID).Preload("Comments").Preload("Likes").Order("created_at desc").Find(&photos).Error; err != nil {
		return nil, err
	}

	return photos, nil
}

func (s *Store) CreateNote(note *models.Note) error {
	return s.db.Create(note).Error
}

func (s *Store) ListVisibleNotesByGroup(groupID uint, now time.Time) ([]models.Note, error) {
	var notes []models.Note
	if err := s.db.Where("group_id = ? AND (show_at IS NULL OR show_at <= ?)", groupID, now).Order("created_at desc").Find(&notes).Error; err != nil {
		return nil, err
	}

	return notes, nil
}

func (s *Store) CreateWishlistItem(item *models.Wishlist) error {
	return s.db.Create(item).Error
}

func (s *Store) ListWishlistByGroup(groupID uint) ([]models.Wishlist, error) {
	var items []models.Wishlist
	if err := s.db.Where("group_id = ?", groupID).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) GetWishlistItemByID(id uint) (*models.Wishlist, error) {
	var item models.Wishlist
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *Store) SaveWishlistItem(item *models.Wishlist) error {
	return s.db.Save(item).Error
}

func (s *Store) EnsureUserByWeChatID(user *models.User) error {
	return s.db.Where(models.User{WeChatID: user.WeChatID}).FirstOrCreate(user).Error
}

func (s *Store) EnsureGroupByInviteCode(group *models.Group) error {
	return s.db.Where(models.Group{InviteCode: group.InviteCode}).FirstOrCreate(group).Error
}
