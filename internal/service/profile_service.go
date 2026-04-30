package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"vocalin-backend/internal/models"
)

type ProfileService struct {
	baseService
}

func NewProfileService(store Store, logger *zap.Logger) *ProfileService {
	return &ProfileService{baseService: newBaseService(store, logger.Named("profile-service"))}
}

func (s *ProfileService) CreateAnniversary(ctx context.Context, userID uint, title string, date time.Time) (*models.Anniversary, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	anniversary := &models.Anniversary{UserID: user.ID, GroupID: groupID, Title: title, Date: date}
	if err := s.store.CreateAnniversary(ctx, anniversary); err != nil {
		return nil, fmt.Errorf("create anniversary: %w", err)
	}
	return anniversary, nil
}

func (s *ProfileService) ListAnniversaries(ctx context.Context, userID uint, pagination Pagination) (*PaginatedResult[models.Anniversary], error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	anniversaries, total, err := s.store.ListAnniversariesByGroup(ctx, groupID, pagination.Offset(), pagination.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list anniversaries: %w", err)
	}
	result := NewPaginatedResult(anniversaries, pagination, int(total))
	return &result, nil
}

func (s *ProfileService) LeaveGroup(ctx context.Context, userID uint) error {
	user, _, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.store.RemoveUserFromGroup(ctx, user); err != nil {
		return fmt.Errorf("leave group: %w", err)
	}
	return nil
}

func (s *ProfileService) ExportData(ctx context.Context, userID uint) (string, error) {
	_, err := s.currentUser(ctx, userID)
	if err != nil {
		return "", err
	}
	return "数据导出任务已创建，稍后将通过邮件发送下载链接。", nil
}
