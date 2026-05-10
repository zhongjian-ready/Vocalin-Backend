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

type AlbumPhotoInput struct {
	URL string
}

type NoteListFilter struct {
	FolderType string
	FolderID   *uint
}

func NewRecordService(store Store, logger *zap.Logger) *RecordService {
	return &RecordService{baseService: newBaseService(store, logger.Named("record-service"))}
}

func (s *RecordService) CreateNoteFolder(ctx context.Context, userID uint, name string) (*models.NoteFolder, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if isReservedNoteFolderName(name) {
		return nil, ErrReservedNoteFolderName
	}
	folder := &models.NoteFolder{GroupID: groupID, OwnerID: user.ID, Name: name}
	if err := s.store.CreateNoteFolder(ctx, folder); err != nil {
		return nil, fmt.Errorf("create note folder: %w", err)
	}
	return folder, nil
}

func (s *RecordService) UpdateNoteFolder(ctx context.Context, userID, folderID uint, name string) (*models.NoteFolder, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	folder, err := s.store.GetNoteFolderByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteFolderNotFound
		}
		return nil, fmt.Errorf("get note folder: %w", err)
	}
	if folder.GroupID != groupID || folder.OwnerID != userID {
		return nil, ErrForbidden
	}
	name = strings.TrimSpace(name)
	if isReservedNoteFolderName(name) {
		return nil, ErrReservedNoteFolderName
	}
	folder.Name = name
	if err := s.store.SaveNoteFolder(ctx, folder); err != nil {
		return nil, fmt.Errorf("update note folder: %w", err)
	}
	return folder, nil
}

func (s *RecordService) DeleteNoteFolder(ctx context.Context, userID, folderID uint) error {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return err
	}
	folder, err := s.store.GetNoteFolderByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteFolderNotFound
		}
		return fmt.Errorf("get note folder: %w", err)
	}
	if folder.GroupID != groupID || folder.OwnerID != userID {
		return ErrForbidden
	}
	if err := s.store.DeleteNoteFolder(ctx, folder.ID, userID); err != nil {
		return fmt.Errorf("delete note folder: %w", err)
	}
	return nil
}

func (s *RecordService) ListNoteFolders(ctx context.Context, userID uint) ([]models.NoteFolder, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	folders, err := s.store.ListNoteFoldersByOwner(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("list note folders: %w", err)
	}
	return folders, nil
}

func (s *RecordService) CreateAlbum(ctx context.Context, userID uint, title, description, visibility string, photos []AlbumPhotoInput) (*models.Album, error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(photos) == 0 {
		return nil, ErrAlbumRequiresPhotos
	}
	album := &models.Album{
		GroupID:     groupID,
		CreatorID:   user.ID,
		Title:       title,
		Description: description,
		Visibility:  normalizeRecordVisibility(visibility),
		Photos:      buildAlbumPhotos(groupID, user.ID, photos),
	}
	if err := s.store.CreateAlbum(ctx, album); err != nil {
		return nil, fmt.Errorf("create album: %w", err)
	}
	return album, nil
}

func (s *RecordService) UpdateAlbum(ctx context.Context, userID, albumID uint, title, description, visibility string, photos []AlbumPhotoInput) (*models.Album, error) {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(photos) == 0 {
		return nil, ErrAlbumRequiresPhotos
	}
	album, err := s.store.GetAlbumByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAlbumNotFound
		}
		return nil, fmt.Errorf("get album: %w", err)
	}
	if album.GroupID != groupID {
		return nil, ErrForbidden
	}
	if album.CreatorID != userID {
		return nil, ErrForbidden
	}
	album.Title = title
	album.Description = description
	album.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SaveAlbum(ctx, album); err != nil {
		return nil, fmt.Errorf("update album: %w", err)
	}
	if err := s.store.ReplaceAlbumPhotos(ctx, album.ID, buildAlbumPhotos(groupID, userID, photos)); err != nil {
		return nil, fmt.Errorf("replace album photos: %w", err)
	}
	updated, err := s.store.GetAlbumByID(ctx, album.ID)
	if err != nil {
		return nil, fmt.Errorf("reload album: %w", err)
	}
	return updated, nil
}

func (s *RecordService) ListAlbums(ctx context.Context, userID uint, pagination Pagination) (*PaginatedResult[models.Album], error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	albums, total, err := s.store.ListAlbumsByGroup(ctx, groupID, user.ID, pagination.Offset(), pagination.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list albums: %w", err)
	}
	result := NewPaginatedResult(albums, pagination, int(total))
	return &result, nil
}

func (s *RecordService) DeleteAlbum(ctx context.Context, userID, albumID uint) error {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return err
	}
	album, err := s.store.GetAlbumByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAlbumNotFound
		}
		return fmt.Errorf("get album: %w", err)
	}
	if album.GroupID != groupID {
		return ErrForbidden
	}
	if album.CreatorID != userID {
		return ErrForbidden
	}
	if err := s.store.DeleteAlbum(ctx, album.ID); err != nil {
		return fmt.Errorf("delete album: %w", err)
	}
	return nil
}

func (s *RecordService) CreateNote(ctx context.Context, userID uint, folderID *uint, content, color, noteType string, showAt *time.Time, visibility string) (*models.Note, error) {
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
	if _, err := s.validateOwnedNoteFolder(ctx, groupID, user.ID, folderID); err != nil {
		return nil, err
	}

	note := &models.Note{GroupID: groupID, AuthorID: user.ID, FolderID: folderID, Content: content, Color: color, Type: noteType, ShowAt: showAt, Visibility: normalizeRecordVisibility(visibility)}
	if err := s.store.CreateNote(ctx, note); err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	created, err := s.store.GetNoteByID(ctx, note.ID)
	if err != nil {
		return nil, fmt.Errorf("reload note: %w", err)
	}
	return created, nil
}

func (s *RecordService) UpdateNote(ctx context.Context, userID, noteID uint, folderID *uint, content, color, noteType string, showAt *time.Time, visibility string) (*models.Note, error) {
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
	if _, err := s.validateOwnedNoteFolder(ctx, groupID, userID, folderID); err != nil {
		return nil, err
	}
	note.Content = content
	note.Color = color
	note.Type = noteType
	note.ShowAt = showAt
	note.FolderID = folderID
	note.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SaveNote(ctx, note); err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	updated, err := s.store.GetNoteByID(ctx, note.ID)
	if err != nil {
		return nil, fmt.Errorf("reload note: %w", err)
	}
	return updated, nil
}

func (s *RecordService) MoveNoteToFolder(ctx context.Context, userID, noteID uint, folderID *uint) (*models.Note, error) {
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
	if _, err := s.validateOwnedNoteFolder(ctx, groupID, userID, folderID); err != nil {
		return nil, err
	}
	note.FolderID = folderID
	if err := s.store.SaveNote(ctx, note); err != nil {
		return nil, fmt.Errorf("move note to folder: %w", err)
	}
	updated, err := s.store.GetNoteByID(ctx, note.ID)
	if err != nil {
		return nil, fmt.Errorf("reload note: %w", err)
	}
	return updated, nil
}

func (s *RecordService) UpdateNoteVisibility(ctx context.Context, userID, noteID uint, visibility string) (*models.Note, error) {
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
	note.Visibility = normalizeRecordVisibility(visibility)
	if err := s.store.SaveNote(ctx, note); err != nil {
		return nil, fmt.Errorf("update note visibility: %w", err)
	}
	updated, err := s.store.GetNoteByID(ctx, note.ID)
	if err != nil {
		return nil, fmt.Errorf("reload note: %w", err)
	}
	return updated, nil
}

func (s *RecordService) ListNotes(ctx context.Context, userID uint, pagination Pagination, filter NoteListFilter) (*PaginatedResult[models.Note], error) {
	user, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if filter.FolderID != nil {
		if _, err := s.validateOwnedNoteFolder(ctx, groupID, user.ID, filter.FolderID); err != nil {
			return nil, err
		}
	}
	notes, total, err := s.store.ListVisibleNotesByGroup(ctx, groupID, user.ID, time.Now(), pagination.Offset(), pagination.PageSize, filter.FolderType, filter.FolderID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	result := NewPaginatedResult(notes, pagination, int(total))
	return &result, nil
}

func (s *RecordService) DeleteNote(ctx context.Context, userID, noteID uint) error {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return err
	}
	note, err := s.store.GetNoteByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteNotFound
		}
		return fmt.Errorf("get note: %w", err)
	}
	if note.GroupID != groupID {
		return ErrForbidden
	}
	if note.AuthorID != userID {
		return ErrForbidden
	}
	if err := s.store.DeleteNote(ctx, note.ID); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return nil
}

func (s *RecordService) validateOwnedNoteFolder(ctx context.Context, groupID uint, userID uint, folderID *uint) (*models.NoteFolder, error) {
	if folderID == nil {
		return nil, nil
	}
	folder, err := s.store.GetNoteFolderByID(ctx, *folderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteFolderNotFound
		}
		return nil, fmt.Errorf("get note folder: %w", err)
	}
	if folder.GroupID != groupID || folder.OwnerID != userID {
		return nil, ErrForbidden
	}
	return folder, nil
}

func isReservedNoteFolderName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return strings.EqualFold(trimmed, "all") || strings.EqualFold(trimmed, "shared")
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

func (s *RecordService) DeleteWishlist(ctx context.Context, userID, itemID uint) error {
	_, groupID, err := s.currentGroupUser(ctx, userID)
	if err != nil {
		return err
	}
	item, err := s.store.GetWishlistItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWishlistItemNotFound
		}
		return fmt.Errorf("get wishlist item: %w", err)
	}
	if item.GroupID != groupID {
		return ErrForbidden
	}
	if !isEditableByUser(item.CreatorID, userID) {
		return ErrForbidden
	}
	if err := s.store.DeleteWishlistItem(ctx, item.ID); err != nil {
		return fmt.Errorf("delete wishlist item: %w", err)
	}
	return nil
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

func buildAlbumPhotos(groupID uint, uploaderID uint, inputs []AlbumPhotoInput) []models.Photo {
	photos := make([]models.Photo, 0, len(inputs))
	for _, input := range inputs {
		photos = append(photos, models.Photo{
			GroupID:    groupID,
			UploaderID: uploaderID,
			URL:        input.URL,
		})
	}
	return photos
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
