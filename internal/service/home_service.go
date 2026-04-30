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
	Group          *models.Group `json:"group"`
	RecentActivity any           `json:"recent_activity,omitempty"`
}

func NewHomeService(store Store, logger *zap.Logger) *HomeService {
	return &HomeService{baseService: newBaseService(store, logger.Named("home-service"))}
}

func (s *HomeService) UpdateTimer(ctx context.Context, userID uint, title string, startDate time.Time) (*models.Group, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.UpdateGroupTimer(ctx, groupID, title, startDate)
	if err != nil {
		return nil, fmt.Errorf("update timer: %w", err)
	}
	return group, nil
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
	return group, nil
}

func (s *HomeService) GetDashboard(ctx context.Context, userID uint) (*DashboardResult, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	group, err := s.store.GetGroupWithMembers(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}

	latestPhoto, photoErr := s.store.GetLatestPhotoByGroup(ctx, group.ID)
	if photoErr != nil && !errors.Is(photoErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load latest photo: %w", photoErr)
	}

	latestNote, noteErr := s.store.GetLatestNoteByGroup(ctx, group.ID)
	if noteErr != nil && !errors.Is(noteErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load latest note: %w", noteErr)
	}

	var recentActivity any
	if latestPhoto != nil && (latestNote == nil || latestPhoto.CreatedAt.After(latestNote.CreatedAt)) {
		recentActivity = map[string]any{"type": "photo", "data": latestPhoto}
	} else if latestNote != nil {
		recentActivity = map[string]any{"type": "note", "data": latestNote}
	}

	return &DashboardResult{Group: group, RecentActivity: recentActivity}, nil
}
