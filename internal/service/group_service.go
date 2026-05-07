package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"vocalin-backend/internal/models"
	"vocalin-backend/pkg/utils"
)

type GroupListItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	InviteCode  string `json:"invite_code"`
	CreatorID   uint   `json:"creator_id"`
	Role        string `json:"role"`
	MemberCount int64  `json:"member_count"`
	IsCurrent   bool   `json:"is_current"`
}

type GroupListResult struct {
	CurrentGroupID *uint           `json:"current_group_id"`
	Groups         []GroupListItem `json:"groups"`
}

type GroupSwitchResult struct {
	CurrentGroupID uint `json:"current_group_id"`
}

type GroupFallbackResult struct {
	CurrentGroupID *uint          `json:"current_group_id"`
	FallbackGroup  *GroupListItem `json:"fallback_group,omitempty"`
}

func (s *GroupService) buildGroupListItem(ctx context.Context, userID uint, group *models.Group, currentGroupID *uint) (*GroupListItem, error) {
	membership, err := s.store.GetGroupMember(ctx, group.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("get group member: %w", err)
	}
	memberCount, err := s.store.CountGroupMembers(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("count group members: %w", err)
	}
	return &GroupListItem{
		ID:          group.ID,
		Name:        group.Name,
		InviteCode:  group.InviteCode,
		CreatorID:   group.CreatorID,
		Role:        membership.Role,
		MemberCount: memberCount,
		IsCurrent:   currentGroupID != nil && *currentGroupID == group.ID,
	}, nil
}

type GroupService struct {
	baseService
}

func NewGroupService(store Store, logger *zap.Logger) *GroupService {
	return &GroupService{baseService: newBaseService(store, logger.Named("group-service"))}
}

func (s *GroupService) CreateGroup(ctx context.Context, userID uint, name string) (*models.Group, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	group := &models.Group{
		Name:       name,
		InviteCode: utils.GenerateInviteCode(6),
		CreatorID:  user.ID,
	}
	if err := s.store.CreateGroupWithCreator(ctx, user, group); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return group, nil
}

func (s *GroupService) JoinGroup(ctx context.Context, userID uint, inviteCode string) (*models.Group, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.GetGroupByInviteCode(ctx, inviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidInviteCode
		}
		return nil, fmt.Errorf("get group by invite code: %w", err)
	}
	if _, err := s.store.GetGroupMember(ctx, group.ID, user.ID); err == nil {
		return nil, ErrUserAlreadyInGroup
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get group member: %w", err)
	}
	if err := s.store.AddUserToGroup(ctx, user, group.ID); err != nil {
		return nil, fmt.Errorf("join group: %w", err)
	}
	return group, nil
}

func (s *GroupService) GetGroupInfo(ctx context.Context, userID uint) (*models.Group, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.GetGroupWithMembers(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group info: %w", err)
	}
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("get group member: %w", err)
	}
	group.MyRole = membership.Role
	return group, nil
}

func (s *GroupService) ListGroups(ctx context.Context, userID uint) (*GroupListResult, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, err := s.store.ListGroupsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	items := make([]GroupListItem, 0, len(groups))
	for _, group := range groups {
		item, err := s.buildGroupListItem(ctx, userID, &group, user.CurrentGroupID)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return &GroupListResult{CurrentGroupID: user.CurrentGroupID, Groups: items}, nil
}

func (s *GroupService) SwitchCurrentGroup(ctx context.Context, userID uint, groupID uint) (*GroupSwitchResult, error) {
	if _, err := s.store.GetGroupMember(ctx, groupID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrForbidden
		}
		return nil, fmt.Errorf("get group member: %w", err)
	}
	if err := s.store.SetCurrentGroup(ctx, userID, &groupID); err != nil {
		return nil, fmt.Errorf("set current group: %w", err)
	}
	return &GroupSwitchResult{CurrentGroupID: groupID}, nil
}

func (s *GroupService) LeaveGroup(ctx context.Context, userID uint, groupID uint) (*GroupFallbackResult, error) {
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrForbidden
		}
		return nil, fmt.Errorf("get group member: %w", err)
	}
	if membership.Role == GroupRoleOwner {
		return nil, ErrGroupOwnershipTransfer
	}
	currentGroupID, err := s.store.RemoveUserFromGroup(ctx, userID, groupID)
	if err != nil {
		return nil, fmt.Errorf("leave group: %w", err)
	}
	result := &GroupFallbackResult{CurrentGroupID: currentGroupID}
	if currentGroupID != nil {
		group, err := s.store.GetGroupWithMembers(ctx, *currentGroupID)
		if err != nil {
			return nil, fmt.Errorf("load current group: %w", err)
		}
		item, err := s.buildGroupListItem(ctx, userID, group, currentGroupID)
		if err != nil {
			return nil, err
		}
		result.FallbackGroup = item
	}
	return result, nil
}

func (s *GroupService) RemoveMember(ctx context.Context, userID uint, groupID uint, targetUserID uint) error {
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrForbidden
		}
		return fmt.Errorf("get group member: %w", err)
	}
	if membership.Role != GroupRoleOwner {
		return ErrGroupOwnerOnly
	}
	if userID == targetUserID {
		return ErrCannotRemoveSelf
	}
	targetMembership, err := s.store.GetGroupMember(ctx, groupID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupMemberNotFound
		}
		return fmt.Errorf("get target member: %w", err)
	}
	if targetMembership.Role == GroupRoleOwner {
		return ErrCannotRemoveGroupOwner
	}
	if _, err := s.store.RemoveUserFromGroup(ctx, targetUserID, groupID); err != nil {
		return fmt.Errorf("remove group member: %w", err)
	}
	return nil
}

func (s *GroupService) TransferOwnership(ctx context.Context, userID uint, groupID uint, targetUserID uint) error {
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrForbidden
		}
		return fmt.Errorf("get group member: %w", err)
	}
	if membership.Role != GroupRoleOwner {
		return ErrGroupOwnerOnly
	}
	if userID == targetUserID {
		return ErrCannotTransferToSelf
	}
	targetMembership, err := s.store.GetGroupMember(ctx, groupID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupMemberNotFound
		}
		return fmt.Errorf("get target member: %w", err)
	}
	if targetMembership.Role == GroupRoleOwner {
		return nil
	}
	if err := s.store.TransferGroupOwnership(ctx, groupID, targetUserID); err != nil {
		return fmt.Errorf("transfer ownership: %w", err)
	}
	return nil
}

func (s *GroupService) DisbandGroup(ctx context.Context, userID uint, groupID uint) (*GroupFallbackResult, error) {
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrForbidden
		}
		return nil, fmt.Errorf("get group member: %w", err)
	}
	if membership.Role != GroupRoleOwner {
		return nil, ErrGroupOwnerOnly
	}
	fallbacks, err := s.store.DisbandGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("disband group: %w", err)
	}
	result := &GroupFallbackResult{CurrentGroupID: fallbacks[userID]}
	if result.CurrentGroupID != nil {
		group, err := s.store.GetGroupWithMembers(ctx, *result.CurrentGroupID)
		if err != nil {
			return nil, fmt.Errorf("load fallback group: %w", err)
		}
		item, err := s.buildGroupListItem(ctx, userID, group, result.CurrentGroupID)
		if err != nil {
			return nil, err
		}
		result.FallbackGroup = item
	}
	return result, nil
}
