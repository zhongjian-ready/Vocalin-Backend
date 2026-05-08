package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"vocalin-backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RecordService struct {
	baseService
}

const (
	recordVisibilityPublic  = "public"
	recordVisibilityPrivate = "private"
)

func NewRecordService(store Store, logger *zap.Logger) *RecordService {
	return &RecordService{baseService: newBaseService(store, logger.Named("record-service"))}
}

func (s *RecordService) CreatePhoto(ctx context.Context, userID uint, url, description, visibility string) (*models.Photo, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	photo := &models.Photo{GroupID: groupID, UploaderID: user.ID, URL: url, Description: description, Visibility: normalizeRecordVisibility(visibility)}
	if err := s.store.CreatePhoto(ctx, photo); err != nil {
		return nil, fmt.Errorf("create photo: %w", err)
	}
	return photo, nil
}

func (s *RecordService) UpdatePhoto(ctx context.Context, userID, photoID uint, url, description, visibility string) (*models.Photo, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	photo, err := s.store.GetPhotoByID(ctx, photoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, fmt.Errorf("get photo: %w", err)
	}
	if photo.GroupID != groupID {
		return nil, ErrForbidden
	}
	if photo.UploaderID != userID {
		return nil, ErrForbidden
	}
	photo.URL = url
	photo.Description = description
	photo.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SavePhoto(ctx, photo); err != nil {
		return nil, fmt.Errorf("update photo: %w", err)
	}
	return photo, nil
}

func (s *RecordService) ListPhotos(ctx context.Context, userID uint, pagination Pagination) (*PaginatedResult[models.Photo], error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	photos, total, err := s.store.ListPhotosByGroup(ctx, groupID, user.ID, pagination.Offset(), pagination.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list photos: %w", err)
	}
	result := NewPaginatedResult(photos, pagination, int(total))
	return &result, nil
}

func (s *RecordService) CreateNote(ctx context.Context, userID uint, content, color, noteType string, showAt *time.Time, visibility string) (*models.Note, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if noteType == "timed" && showAt == nil {
		return nil, ErrTimedNoteRequiresShowAt
	}
	if showAt != nil && noteType != "timed" {
		showAt = nil
	}
	showAt = normalizeTimeToLocal(showAt)

	note := &models.Note{GroupID: groupID, AuthorID: user.ID, Content: content, Color: color, Type: noteType, ShowAt: showAt, Visibility: normalizeRecordVisibility(visibility)}
	if err := s.store.CreateNote(ctx, note); err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	return note, nil
}

func (s *RecordService) UpdateNote(ctx context.Context, userID, noteID uint, content, color, noteType string, showAt *time.Time, visibility string) (*models.Note, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	note, err := s.store.GetNoteByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	if note.GroupID != groupID {
		return nil, ErrForbidden
	}
	if note.AuthorID != userID {
		return nil, ErrForbidden
	}
	if noteType == "timed" && showAt == nil {
		return nil, ErrTimedNoteRequiresShowAt
	}
	if showAt != nil && noteType != "timed" {
		showAt = nil
	}
	showAt = normalizeTimeToLocal(showAt)
	note.Content = content
	note.Color = color
	note.Type = noteType
	note.ShowAt = showAt
	note.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SaveNote(ctx, note); err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	return note, nil
}

func (s *RecordService) ListNotes(ctx context.Context, userID uint, pagination Pagination) (*PaginatedResult[models.Note], error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	notes, total, err := s.store.ListVisibleNotesByGroup(ctx, groupID, user.ID, time.Now(), pagination.Offset(), pagination.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	result := NewPaginatedResult(notes, pagination, int(total))
	return &result, nil
}

func (s *RecordService) CreateWishlist(ctx context.Context, userID uint, content string, priority string, visibility string) (*models.Wishlist, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	item := &models.Wishlist{GroupID: groupID, CreatorID: user.ID, Content: content, Priority: normalizeWishlistPriority(priority), Visibility: normalizeRecordVisibility(visibility)}
	if err := s.store.CreateWishlistItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create wishlist item: %w", err)
	}
	return item, nil
}

func (s *RecordService) UpdateWishlist(ctx context.Context, userID, itemID uint, content, priority, visibility string) (*models.Wishlist, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	item, err := s.store.GetWishlistItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWishlistItemNotFound
		}
		return nil, fmt.Errorf("get wishlist item: %w", err)
	}
	if item.GroupID != groupID {
		return nil, ErrForbidden
	}
	if !isEditableByUser(item.CreatorID, userID) {
		return nil, ErrForbidden
	}
	item.Content = content
	item.Priority = normalizeWishlistPriority(priority)
	item.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SaveWishlistItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update wishlist item: %w", err)
	}
	return item, nil
}

func (s *RecordService) ListWishlist(ctx context.Context, userID uint, pagination Pagination) (*PaginatedResult[models.Wishlist], error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.store.ListWishlistByGroup(ctx, groupID, user.ID, pagination.Offset(), pagination.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list wishlist: %w", err)
	}
	result := NewPaginatedResult(items, pagination, int(total))
	return &result, nil
}

func (s *RecordService) CompleteWishlist(ctx context.Context, userID, itemID uint) (*models.Wishlist, error) {
	return s.setWishlistCompletion(ctx, userID, itemID, true)
}

func (s *RecordService) IncompleteWishlist(ctx context.Context, userID, itemID uint) (*models.Wishlist, error) {
	return s.setWishlistCompletion(ctx, userID, itemID, false)
}

func (s *RecordService) setWishlistCompletion(ctx context.Context, userID, itemID uint, completed bool) (*models.Wishlist, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	item, err := s.store.GetWishlistItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWishlistItemNotFound
		}
		return nil, fmt.Errorf("get wishlist item: %w", err)
	}
	if item.GroupID != groupID {
		return nil, ErrForbidden
	}
	if item.Visibility == recordVisibilityPrivate && !isEditableByUser(item.CreatorID, userID) {
		return nil, ErrForbidden
	}
	item.IsCompleted = completed
	if completed {
		now := time.Now()
		item.CompletedAt = &now
	} else {
		item.CompletedAt = nil
	}
	if err := s.store.SaveWishlistItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update wishlist item completion: %w", err)
	}
	return item, nil
}

func normalizeWishlistPriority(priority string) string {
	normalized := strings.ToLower(strings.TrimSpace(priority))
	if normalized == "" {
		return "medium"
	}
	return normalized
}

func normalizeRecordVisibility(visibility string) string {
	normalized := strings.ToLower(strings.TrimSpace(visibility))
	if normalized == "" {
		return recordVisibilityPublic
	}
	return normalized
}

func isEditableByUser(ownerID uint, userID uint) bool {
	return ownerID == 0 || ownerID == userID
}

func normalizeTimeToLocal(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.In(time.Local)
	return &normalized
}
