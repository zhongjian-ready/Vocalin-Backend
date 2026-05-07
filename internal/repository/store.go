package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"vocalin-backend/internal/models"
)

const (
	groupRoleOwner  = "owner"
	groupRoleMember = "member"
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

func (s *Store) GetGroupByID(ctx context.Context, groupID uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) GetGroupMember(ctx context.Context, groupID uint, userID uint) (*models.GroupMember, error) {
	var membership models.GroupMember
	if err := s.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).First(&membership).Error; err != nil {
		return nil, err
	}
	return &membership, nil
}

func (s *Store) ListGroupMembersByUser(ctx context.Context, userID uint) ([]models.GroupMember, error) {
	var memberships []models.GroupMember
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at asc, id asc").Find(&memberships).Error; err != nil {
		return nil, err
	}
	return memberships, nil
}

func (s *Store) SetCurrentGroup(ctx context.Context, userID uint, groupID *uint) error {
	return s.updateUserGroupID(s.db.WithContext(ctx), userID, groupID)
}

func (s *Store) ListGroupsByUser(ctx context.Context, userID uint) ([]models.Group, error) {
	var groups []models.Group
	if err := s.db.WithContext(ctx).
		Model(&models.Group{}).
		Joins("JOIN group_members ON group_members.group_id = groups.id AND group_members.deleted_at IS NULL").
		Where("group_members.user_id = ?", userID).
		Order("group_members.created_at asc, group_members.id asc").
		Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *Store) CountGroupMembers(ctx context.Context, groupID uint) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.GroupMember{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) GetFirstGroupByUser(ctx context.Context, userID uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).
		Model(&models.Group{}).
		Joins("JOIN group_members ON group_members.group_id = groups.id AND group_members.deleted_at IS NULL").
		Where("group_members.user_id = ?", userID).
		Order("group_members.created_at asc, group_members.id asc").
		First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
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
		if err := tx.Create(&models.GroupMember{UserID: user.ID, GroupID: group.ID, Role: groupRoleOwner}).Error; err != nil {
			return err
		}
		user.CurrentGroupID = &group.ID
		return s.updateUserGroupID(tx, user.ID, user.CurrentGroupID)
	})
}

func (s *Store) AddUserToGroup(ctx context.Context, user *models.User, groupID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership models.GroupMember
		err := tx.Unscoped().Where("user_id = ? AND group_id = ?", user.ID, groupID).First(&membership).Error
		switch {
		case err == nil:
			if err := tx.Unscoped().Model(&membership).Updates(map[string]any{
				"deleted_at": nil,
				"role":       groupRoleMember,
			}).Error; err != nil {
				return err
			}
		case err == gorm.ErrRecordNotFound:
			if err := tx.Create(&models.GroupMember{UserID: user.ID, GroupID: groupID, Role: groupRoleMember}).Error; err != nil {
				return err
			}
		default:
			return err
		}
		user.CurrentGroupID = &groupID
		return s.updateUserGroupID(tx, user.ID, user.CurrentGroupID)
	})
}

func (s *Store) RemoveUserFromGroup(ctx context.Context, userID uint, groupID uint) (*uint, error) {
	var nextGroupID *uint
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND group_id = ?", userID, groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		if user.CurrentGroupID == nil || *user.CurrentGroupID != groupID {
			nextGroupID = user.CurrentGroupID
			return nil
		}
		var first models.GroupMember
		err := tx.Where("user_id = ?", userID).Order("created_at asc, id asc").First(&first).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return s.updateUserGroupID(tx, userID, nil)
			}
			return err
		}
		nextGroupID = &first.GroupID
		return s.updateUserGroupID(tx, userID, nextGroupID)
	})
	if err != nil {
		return nil, err
	}
	return nextGroupID, nil
}

func (s *Store) TransferGroupOwnership(ctx context.Context, groupID uint, targetUserID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Group{}).Where("id = ?", groupID).Update("creator_id", targetUserID).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.GroupMember{}).Where("group_id = ?", groupID).Update("role", groupRoleMember).Error; err != nil {
			return err
		}
		return tx.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, targetUserID).Update("role", groupRoleOwner).Error
	})
}

func (s *Store) CreateGroupRequest(ctx context.Context, request *models.GroupRequest) error {
	return s.db.WithContext(ctx).Create(request).Error
}

func (s *Store) GetGroupRequestByID(ctx context.Context, requestID uint) (*models.GroupRequest, error) {
	var request models.GroupRequest
	if err := s.db.WithContext(ctx).First(&request, requestID).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *Store) FindPendingJoinRequest(ctx context.Context, groupID uint, requesterUserID uint) (*models.GroupRequest, error) {
	var request models.GroupRequest
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND requester_user_id = ? AND type = ? AND status = ?", groupID, requesterUserID, models.GroupRequestTypeJoin, models.GroupRequestStatusPending).
		First(&request).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *Store) FindPendingOwnershipTransferRequest(ctx context.Context, groupID uint) (*models.GroupRequest, error) {
	var request models.GroupRequest
	if err := s.db.WithContext(ctx).
		Where("group_id = ? AND type = ? AND status = ?", groupID, models.GroupRequestTypeOwnershipTransfer, models.GroupRequestStatusPending).
		Order("created_at desc, id desc").
		First(&request).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (s *Store) ListPendingGroupRequestsByRequester(ctx context.Context, requesterUserID uint) ([]models.GroupRequest, error) {
	var requests []models.GroupRequest
	if err := s.db.WithContext(ctx).
		Where("requester_user_id = ? AND status = ?", requesterUserID, models.GroupRequestStatusPending).
		Order("created_at desc, id desc").
		Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *Store) ListPendingGroupRequestsForTarget(ctx context.Context, targetUserID uint) ([]models.GroupRequest, error) {
	var requests []models.GroupRequest
	if err := s.db.WithContext(ctx).
		Where("target_user_id = ? AND status = ?", targetUserID, models.GroupRequestStatusPending).
		Order("created_at desc, id desc").
		Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *Store) CountPendingGroupRequestsForTarget(ctx context.Context, targetUserID uint) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&models.GroupRequest{}).
		Where("target_user_id = ? AND status = ?", targetUserID, models.GroupRequestStatusPending).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) ApproveGroupRequest(ctx context.Context, requestID uint, reviewerUserID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request models.GroupRequest
		if err := tx.Where("id = ? AND status = ?", requestID, models.GroupRequestStatusPending).First(&request).Error; err != nil {
			return err
		}

		now := time.Now()
		switch request.Type {
		case models.GroupRequestTypeJoin:
			if err := s.upsertGroupMembership(tx, request.RequesterUserID, request.GroupID, groupRoleMember, true); err != nil {
				return err
			}
		case models.GroupRequestTypeOwnershipTransfer:
			if err := tx.Model(&models.Group{}).Where("id = ?", request.GroupID).Update("creator_id", request.TargetUserID).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.GroupMember{}).Where("group_id = ?", request.GroupID).Update("role", groupRoleMember).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", request.GroupID, request.TargetUserID).Update("role", groupRoleOwner).Error; err != nil {
				return err
			}
		default:
			return gorm.ErrInvalidData
		}

		return tx.Model(&models.GroupRequest{}).Where("id = ?", request.ID).Updates(map[string]any{
			"status":           models.GroupRequestStatusApproved,
			"reviewed_at":      &now,
			"reviewer_user_id": reviewerUserID,
		}).Error
	})
}

func (s *Store) RejectGroupRequest(ctx context.Context, requestID uint, reviewerUserID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request models.GroupRequest
		if err := tx.Where("id = ? AND status = ?", requestID, models.GroupRequestStatusPending).First(&request).Error; err != nil {
			return err
		}

		now := time.Now()
		return tx.Model(&models.GroupRequest{}).Where("id = ?", request.ID).Updates(map[string]any{
			"status":           models.GroupRequestStatusRejected,
			"reviewed_at":      &now,
			"reviewer_user_id": reviewerUserID,
		}).Error
	})
}

func (s *Store) DisbandGroup(ctx context.Context, groupID uint) (map[uint]*uint, error) {
	fallbacks := make(map[uint]*uint)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var memberships []models.GroupMember
		if err := tx.Where("group_id = ?", groupID).Find(&memberships).Error; err != nil {
			return err
		}
		currentGroups := make(map[uint]*uint, len(memberships))
		for _, membership := range memberships {
			var user models.User
			if err := tx.Select("id", "group_id").First(&user, membership.UserID).Error; err != nil {
				return err
			}
			currentGroups[membership.UserID] = user.CurrentGroupID
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
			return err
		}
		for _, membership := range memberships {
			if currentGroups[membership.UserID] != nil && *currentGroups[membership.UserID] != groupID {
				fallbacks[membership.UserID] = currentGroups[membership.UserID]
				continue
			}
			var next models.GroupMember
			err := tx.Where("user_id = ?", membership.UserID).Order("created_at asc, id asc").First(&next).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				fallbacks[membership.UserID] = nil
				if err := s.updateUserGroupID(tx, membership.UserID, nil); err != nil {
					return err
				}
				continue
			}
			nextGroupID := next.GroupID
			fallbacks[membership.UserID] = &nextGroupID
			if err := s.updateUserGroupID(tx, membership.UserID, &nextGroupID); err != nil {
				return err
			}
		}
		return tx.Delete(&models.Group{}, groupID).Error
	})
	if err != nil {
		return nil, err
	}
	return fallbacks, nil
}

func (s *Store) updateUserGroupID(db *gorm.DB, userID uint, groupID *uint) error {
	return db.Model(&models.User{}).Where("id = ?", userID).Update("group_id", groupID).Error
}

func (s *Store) upsertGroupMembership(tx *gorm.DB, userID uint, groupID uint, role string, setCurrentIfEmpty bool) error {
	var membership models.GroupMember
	err := tx.Unscoped().Where("user_id = ? AND group_id = ?", userID, groupID).First(&membership).Error
	switch {
	case err == nil:
		if err := tx.Unscoped().Model(&membership).Updates(map[string]any{
			"deleted_at": nil,
			"role":       role,
		}).Error; err != nil {
			return err
		}
	case err == gorm.ErrRecordNotFound:
		if err := tx.Create(&models.GroupMember{UserID: userID, GroupID: groupID, Role: role}).Error; err != nil {
			return err
		}
	default:
		return err
	}

	if !setCurrentIfEmpty {
		return nil
	}

	var user models.User
	if err := tx.Select("id", "group_id").First(&user, userID).Error; err != nil {
		return err
	}
	if user.CurrentGroupID != nil {
		return nil
	}
	return s.updateUserGroupID(tx, userID, &groupID)
}

func (s *Store) GetGroupWithMembers(ctx context.Context, groupID uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return nil, err
	}
	members, err := s.getUsersByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	group.Members = members
	return &group, nil
}

func (s *Store) getUsersByGroup(ctx context.Context, groupID uint) ([]models.User, error) {
	var users []models.User
	if err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Select("users.*, group_members.role AS group_role").
		Joins("JOIN group_members ON group_members.user_id = users.id AND group_members.deleted_at IS NULL").
		Where("group_members.group_id = ?", groupID).
		Order("group_members.created_at asc, group_members.id asc").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Store) UpdateGroupTimer(ctx context.Context, groupID uint, title string, startDate time.Time) (*models.Group, error) {
	var group models.Group
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return nil, err
	}
	group.TimerTitle = title
	group.TimerStartDate = &startDate
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
