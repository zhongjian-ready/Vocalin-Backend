package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	CurrentGroupID  *uint                     `json:"current_group_id"`
	Groups          []GroupListItem           `json:"groups"`
	PendingRequests []PendingGroupRequestItem `json:"pending_requests,omitempty"`
}

type PendingGroupRequestItem struct {
	ID           uint      `json:"id"`
	GroupID      uint      `json:"group_id"`
	GroupName    string    `json:"group_name"`
	InviteCode   string    `json:"invite_code"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	TargetUserID uint      `json:"target_user_id"`
}

type GroupSwitchResult struct {
	CurrentGroupID uint `json:"current_group_id"`
}

type GroupFallbackResult struct {
	CurrentGroupID *uint          `json:"current_group_id"`
	FallbackGroup  *GroupListItem `json:"fallback_group,omitempty"`
}

const maxGroupMembers int64 = 24

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
	membership, err := s.store.GetGroupMember(ctx, group.ID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get creator group member: %w", err)
	}
	applyGroupMembershipMetadata(group, membership)
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
	memberCount, err := s.store.CountGroupMembers(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("count group members: %w", err)
	}
	if memberCount >= maxGroupMembers {
		return nil, ErrGroupMemberLimitReached
	}
	if _, err := s.store.FindPendingJoinRequest(ctx, group.ID, user.ID); err == nil {
		return nil, ErrGroupJoinRequestPending
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find pending join request: %w", err)
	}
	request := &models.GroupRequest{
		GroupID:         group.ID,
		RequesterUserID: user.ID,
		TargetUserID:    group.CreatorID,
		Type:            models.GroupRequestTypeJoin,
		Status:          models.GroupRequestStatusPending,
	}
	if err := s.store.CreateGroupRequest(ctx, request); err != nil {
		return nil, fmt.Errorf("create join request: %w", err)
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
	applyGroupMembershipMetadata(group, membership)
	if err := s.attachPendingOwnershipTransfer(ctx, userID, group); err != nil {
		return nil, err
	}
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
	pendingRequests, err := s.store.ListPendingGroupRequestsByRequester(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list pending group requests: %w", err)
	}
	pendingItems, err := s.buildPendingGroupRequestItems(ctx, pendingRequests)
	if err != nil {
		return nil, err
	}
	return &GroupListResult{CurrentGroupID: user.CurrentGroupID, Groups: items, PendingRequests: pendingItems}, nil
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
	if _, err := s.store.FindPendingOwnershipTransferRequest(ctx, groupID); err == nil {
		return ErrGroupTransferPending
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find pending ownership transfer: %w", err)
	}
	request := &models.GroupRequest{
		GroupID:         groupID,
		RequesterUserID: userID,
		TargetUserID:    targetUserID,
		Type:            models.GroupRequestTypeOwnershipTransfer,
		Status:          models.GroupRequestStatusPending,
	}
	if err := s.store.CreateGroupRequest(ctx, request); err != nil {
		return fmt.Errorf("create ownership transfer request: %w", err)
	}
	return nil
}

func (s *GroupService) buildPendingGroupRequestItems(ctx context.Context, requests []models.GroupRequest) ([]PendingGroupRequestItem, error) {
	items := make([]PendingGroupRequestItem, 0, len(requests))
	for _, request := range requests {
		group, err := s.store.GetGroupByID(ctx, request.GroupID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, fmt.Errorf("get pending request group: %w", err)
		}
		items = append(items, PendingGroupRequestItem{
			ID:           request.ID,
			GroupID:      request.GroupID,
			GroupName:    group.Name,
			InviteCode:   group.InviteCode,
			Type:         request.Type,
			Status:       request.Status,
			CreatedAt:    request.CreatedAt,
			TargetUserID: request.TargetUserID,
		})
	}
	return items, nil
}

func (s *GroupService) attachPendingOwnershipTransfer(ctx context.Context, userID uint, group *models.Group) error {
	request, err := s.store.FindPendingOwnershipTransferRequest(ctx, group.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("find pending ownership transfer: %w", err)
	}
	if request.RequesterUserID != userID {
		return nil
	}
	group.PendingOwnershipTransfer = true
	group.PendingOwnershipTransferRequestID = &request.ID
	group.PendingOwnershipTransferToUserID = &request.TargetUserID
	return nil
}

func (s *GroupService) ReviewJoinRequest(ctx context.Context, userID uint, groupID uint, requestID uint, action string) error {
	request, err := s.store.GetGroupRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupRequestNotFound
		}
		return fmt.Errorf("get group request: %w", err)
	}
	if request.GroupID != groupID || request.Type != models.GroupRequestTypeJoin {
		return ErrGroupRequestNotFound
	}
	return s.reviewGroupRequest(ctx, userID, request, action)
}

func (s *GroupService) ReviewOwnershipTransfer(ctx context.Context, userID uint, groupID uint, action string) error {
	request, err := s.store.FindPendingOwnershipTransferRequest(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupRequestNotFound
		}
		return fmt.Errorf("find pending ownership transfer: %w", err)
	}
	if request.Type != models.GroupRequestTypeOwnershipTransfer {
		return ErrGroupRequestNotFound
	}
	return s.reviewGroupRequest(ctx, userID, request, action)
}

func (s *GroupService) reviewGroupRequest(ctx context.Context, userID uint, request *models.GroupRequest, action string) error {
	if request.TargetUserID != userID {
		return ErrForbidden
	}
	if request.Status != models.GroupRequestStatusPending {
		return ErrGroupRequestHandled
	}

	switch action {
	case "approve":
		if err := s.store.ApproveGroupRequest(ctx, request.ID, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGroupRequestHandled
			}
			if errors.Is(err, gorm.ErrInvalidData) {
				return ErrGroupMemberLimitReached
			}
			return fmt.Errorf("approve group request: %w", err)
		}
	case "reject":
		if err := s.store.RejectGroupRequest(ctx, request.ID, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGroupRequestHandled
			}
			return fmt.Errorf("reject group request: %w", err)
		}
	default:
		return ErrForbidden
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
