package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"vocalin-backend/internal/models"
)

type HomeService struct {
	baseService
}

type DashboardResult struct {
	Group               *models.Group `json:"group"`
	RecentActivity      any           `json:"recent_activity,omitempty"`
	PendingMessageCount int64         `json:"pending_message_count"`
}

type MessageListItem struct {
	ID                uint      `json:"id"`
	GroupID           uint      `json:"group_id"`
	GroupName         string    `json:"group_name"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	RequesterUserID   uint      `json:"requester_user_id"`
	RequesterNickname string    `json:"requester_nickname"`
	TargetUserID      uint      `json:"target_user_id"`
	TargetNickname    string    `json:"target_nickname"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewHomeService(store Store, logger *zap.Logger) *HomeService {
	return &HomeService{baseService: newBaseService(store, logger.Named("home-service"))}
}

func (s *HomeService) UpdateStatus(ctx context.Context, userID uint, status string) (*models.User, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpdateUserStatus(ctx, user, status, time.Now()); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	return user, nil
}

func (s *HomeService) UpdatePinnedMessage(ctx context.Context, userID uint, content string) (*models.Group, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.UpdatePinnedMessage(ctx, groupID, user.ID, content)
	if err != nil {
		return nil, fmt.Errorf("update pinned message: %w", err)
	}
	membership, err := s.store.GetGroupMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("get group member: %w", err)
	}
	applyGroupMembershipMetadata(group, membership)
	return group, nil
}

func (s *HomeService) GetDashboard(ctx context.Context, userID uint) (*DashboardResult, error) {
	user, err := s.currentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	pendingMessageCount, err := s.store.CountPendingGroupRequestsForTarget(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count pending group requests: %w", err)
	}
	if user.CurrentGroupID == nil {
		return &DashboardResult{PendingMessageCount: pendingMessageCount}, nil
	}
	membership, err := s.store.GetGroupMember(ctx, *user.CurrentGroupID, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &DashboardResult{PendingMessageCount: pendingMessageCount}, nil
		}
		return nil, fmt.Errorf("get group member: %w", err)
	}
	groupID := *user.CurrentGroupID
	group, err := s.store.GetGroupWithMembers(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &DashboardResult{PendingMessageCount: pendingMessageCount}, nil
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	applyGroupMembershipMetadata(group, membership)
	if request, err := s.store.FindPendingOwnershipTransferRequest(ctx, groupID); err == nil {
		if request.RequesterUserID == userID {
			group.PendingOwnershipTransfer = true
			group.PendingOwnershipTransferRequestID = &request.ID
			group.PendingOwnershipTransferToUserID = &request.TargetUserID
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find pending ownership transfer: %w", err)
	}

	latestAlbum, albumErr := s.store.GetLatestVisibleAlbumByGroup(ctx, group.ID, userID)
	if albumErr != nil && !errors.Is(albumErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load latest album: %w", albumErr)
	}

	latestNote, noteErr := s.store.GetLatestVisibleNoteByGroup(ctx, group.ID, userID, time.Now())
	if noteErr != nil && !errors.Is(noteErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load latest note: %w", noteErr)
	}

	var recentActivity any
	if latestAlbum != nil && (latestNote == nil || latestAlbum.CreatedAt.After(latestNote.CreatedAt)) {
		recentActivity = map[string]any{"type": "album", "data": latestAlbum}
	} else if latestNote != nil {
		recentActivity = map[string]any{"type": "note", "data": latestNote}
	}

	return &DashboardResult{Group: group, RecentActivity: recentActivity, PendingMessageCount: pendingMessageCount}, nil
}

func (s *HomeService) ListMessages(ctx context.Context, userID uint) ([]MessageListItem, error) {
	requests, err := s.store.ListPendingGroupRequestsForTarget(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list pending messages: %w", err)
	}
	items := make([]MessageListItem, 0, len(requests))
	for _, request := range requests {
		group, err := s.store.GetGroupByID(ctx, request.GroupID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, fmt.Errorf("get message group: %w", err)
		}
		requester, err := s.store.GetUserByID(ctx, request.RequesterUserID)
		if err != nil {
			return nil, fmt.Errorf("get requester: %w", err)
		}
		target, err := s.store.GetUserByID(ctx, request.TargetUserID)
		if err != nil {
			return nil, fmt.Errorf("get target user: %w", err)
		}
		items = append(items, MessageListItem{
			ID:                request.ID,
			GroupID:           request.GroupID,
			GroupName:         group.Name,
			Type:              request.Type,
			Status:            request.Status,
			RequesterUserID:   request.RequesterUserID,
			RequesterNickname: requester.Nickname,
			TargetUserID:      request.TargetUserID,
			TargetNickname:    target.Nickname,
			CreatedAt:         request.CreatedAt,
		})
	}
	return items, nil
}
