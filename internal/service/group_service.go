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
	if user.GroupID != nil {
		return nil, ErrUserAlreadyInGroup
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
	if user.GroupID != nil {
		return nil, ErrUserAlreadyInGroup
	}

	group, err := s.store.GetGroupByInviteCode(ctx, inviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidInviteCode
		}
		return nil, fmt.Errorf("get group by invite code: %w", err)
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
	return group, nil
}
