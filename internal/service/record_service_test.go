package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

func TestRecordServiceListAlbumsWithPagination(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "wechat-photo-user", Nickname: "photo-user", Phone: "13800138001", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "g1", InviteCode: "PHOTO1", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	for index := 0; index < 3; index++ {
		album := &models.Album{
			GroupID:     group.ID,
			CreatorID:   user.ID,
			Title:       "album",
			Description: "album",
			Visibility:  "public",
			Photos: []models.Photo{{
				GroupID:    group.ID,
				UploaderID: user.ID,
				URL:        "https://example.com/photo.jpg",
			}},
		}
		if err := store.CreateAlbum(ctx, album); err != nil {
			t.Fatalf("create album: %v", err)
		}
	}

	result, err := svc.ListAlbums(ctx, user.ID, NewPagination(1, 2))
	if err != nil {
		t.Fatalf("list albums: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(result.Items))
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}
	if result.TotalPages != 2 {
		t.Fatalf("expected total pages 2, got %d", result.TotalPages)
	}
}

func TestRecordServiceIncompleteWishlist(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	group := &models.Group{Name: "wishlist-group", InviteCode: "WISH01"}
	if err := store.CreateGroup(ctx, group); err != nil {
		t.Fatalf("create group: %v", err)
	}

	user := &models.User{WeChatID: "wishlist-user", Nickname: "wishlist-user", Phone: "13800138002", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.AddUserToGroup(ctx, user, group.ID); err != nil {
		t.Fatalf("add user to group: %v", err)
	}

	completedAt := time.Now().Add(-time.Hour)
	item := &models.Wishlist{GroupID: group.ID, Content: "wish", IsCompleted: true, CompletedAt: &completedAt}
	if err := store.CreateWishlistItem(ctx, item); err != nil {
		t.Fatalf("create wishlist item: %v", err)
	}

	updated, err := svc.IncompleteWishlist(ctx, user.ID, item.ID)
	if err != nil {
		t.Fatalf("incomplete wishlist: %v", err)
	}
	if updated.IsCompleted {
		t.Fatal("expected wishlist item to be marked incomplete")
	}
	if updated.CompletedAt != nil {
		t.Fatalf("expected completed_at to be cleared, got %v", updated.CompletedAt)
	}

	reloaded, err := store.GetWishlistItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("reload wishlist item: %v", err)
	}
	if reloaded.IsCompleted {
		t.Fatal("expected persisted wishlist item to be incomplete")
	}
	if reloaded.CompletedAt != nil {
		t.Fatalf("expected persisted completed_at to be nil, got %v", reloaded.CompletedAt)
	}
}

func TestRecordServiceCreateWishlistPersistsPriority(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "wishlist-priority-user", Nickname: "wishlist-priority-user", Phone: "13800138003", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "wishlist-priority-group", InviteCode: "WISH02", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	created, err := svc.CreateWishlist(ctx, user.ID, "plan a trip", "high", "public")
	if err != nil {
		t.Fatalf("create wishlist: %v", err)
	}
	if created.Priority != "high" {
		t.Fatalf("expected created priority high, got %q", created.Priority)
	}

	result, err := svc.ListWishlist(ctx, user.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list wishlist: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 wishlist item, got %d", len(result.Items))
	}
	if result.Items[0].Priority != "high" {
		t.Fatalf("expected listed priority high, got %q", result.Items[0].Priority)
	}

	defaulted, err := svc.CreateWishlist(ctx, user.ID, "buy flowers", "", "")
	if err != nil {
		t.Fatalf("create wishlist with default priority: %v", err)
	}
	if defaulted.Priority != "medium" {
		t.Fatalf("expected default priority medium, got %q", defaulted.Priority)
	}
	stored, err := store.GetWishlistItemByID(ctx, defaulted.ID)
	if err != nil {
		t.Fatalf("reload wishlist item: %v", err)
	}
	if stored.Priority != "medium" {
		t.Fatalf("expected stored default priority medium, got %q", stored.Priority)
	}
}

func TestRecordServiceAlbumVisibilityCreateUpdateAndList(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "photo-owner", Nickname: "photo-owner", Phone: "13800138005", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	viewer := &models.User{WeChatID: "photo-viewer", Nickname: "photo-viewer", Phone: "13800138006", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, viewer); err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	group := &models.Group{Name: "photo-visibility-group", InviteCode: "PHOTO2", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, viewer, group.ID); err != nil {
		t.Fatalf("add viewer to group: %v", err)
	}

	album, err := svc.CreateAlbum(ctx, owner.ID, "travel", "secret", "private", []AlbumPhotoInput{{
		URL: "https://example.com/private-photo.jpg",
	}})
	if err != nil {
		t.Fatalf("create album: %v", err)
	}
	if album.Visibility != "private" {
		t.Fatalf("expected private visibility, got %q", album.Visibility)
	}
	if len(album.Photos) != 1 {
		t.Fatalf("expected one camera photo in album, got %#v", album.Photos)
	}

	result, err := svc.ListAlbums(ctx, viewer.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list albums for viewer: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected private album to be hidden, got %d items", len(result.Items))
	}
	updated, err := svc.UpdateAlbum(ctx, owner.ID, album.ID, "travel shared", "shared", "public", []AlbumPhotoInput{{
		URL: "https://example.com/public-photo.jpg",
	}})
	if err != nil {
		t.Fatalf("update album: %v", err)
	}
	if updated.Visibility != "public" {
		t.Fatalf("expected public visibility after update, got %q", updated.Visibility)
	}

	result, err = svc.ListAlbums(ctx, viewer.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list albums after update: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 visible album after update, got %d", len(result.Items))
	}
}

func TestRecordServiceNoteVisibilityCreateUpdateAndList(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "note-owner", Nickname: "note-owner", Phone: "13800138007", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	viewer := &models.User{WeChatID: "note-viewer", Nickname: "note-viewer", Phone: "13800138008", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, viewer); err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	group := &models.Group{Name: "note-visibility-group", InviteCode: "NOTE10", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, viewer, group.ID); err != nil {
		t.Fatalf("add viewer to group: %v", err)
	}

	folder, err := svc.CreateNoteFolder(ctx, owner.ID, "Diary")
	if err != nil {
		t.Fatalf("create note folder: %v", err)
	}

	note, err := svc.CreateNote(ctx, owner.ID, &folder.ID, "secret note", "#fff", "normal", nil, "private")
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if note.Visibility != "private" {
		t.Fatalf("expected private visibility, got %q", note.Visibility)
	}
	if note.Folder == nil || note.Folder.Name != "Diary" {
		t.Fatalf("expected folder Diary, got %+v", note.Folder)
	}

	result, err := svc.ListNotes(ctx, viewer.ID, NewPagination(1, 10), NoteListFilter{})
	if err != nil {
		t.Fatalf("list notes for viewer: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected private note to be hidden, got %d items", len(result.Items))
	}

	updated, err := svc.UpdateNote(ctx, owner.ID, note.ID, &folder.ID, "shared note", "#000", "normal", nil, "public")
	if err != nil {
		t.Fatalf("update note: %v", err)
	}
	if updated.Visibility != "public" {
		t.Fatalf("expected public visibility after update, got %q", updated.Visibility)
	}

	result, err = svc.ListNotes(ctx, viewer.ID, NewPagination(1, 10), NoteListFilter{})
	if err != nil {
		t.Fatalf("list notes after update: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 visible note after update, got %d", len(result.Items))
	}
	if result.Items[0].Folder == nil || result.Items[0].Folder.Name != "Diary" {
		t.Fatalf("expected visible note folder Diary, got %+v", result.Items[0].Folder)
	}
}

func TestRecordServiceDeleteNote(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "delete-note-user", Nickname: "delete-note-user", Phone: "13800138012", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "delete-note-group", InviteCode: "NOTE11", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	note := &models.Note{GroupID: group.ID, AuthorID: user.ID, Content: "delete me", Type: "normal"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	if err := svc.DeleteNote(ctx, user.ID, note.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}

	_, err := store.GetNoteByID(ctx, note.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted note to be missing, got %v", err)
	}
}

func TestRecordServiceNoteFolderCRUDAndDeleteFallback(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "note-folder-user", Nickname: "note-folder-user", Phone: "13800138013", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "note-folder-group", InviteCode: "NOTE13", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group: %v", err)
	}

	folder, err := svc.CreateNoteFolder(ctx, user.ID, "Work")
	if err != nil {
		t.Fatalf("create note folder: %v", err)
	}
	if folder.Name != "Work" {
		t.Fatalf("expected folder name Work, got %q", folder.Name)
	}

	if _, err := svc.CreateNoteFolder(ctx, user.ID, "shared"); !errors.Is(err, ErrReservedNoteFolderName) {
		t.Fatalf("expected reserved folder error, got %v", err)
	}

	note, err := svc.CreateNote(ctx, user.ID, &folder.ID, "classified", "#fff", "normal", nil, "private")
	if err != nil {
		t.Fatalf("create note with folder: %v", err)
	}

	updated, err := svc.UpdateNoteFolder(ctx, user.ID, folder.ID, "Travel")
	if err != nil {
		t.Fatalf("update note folder: %v", err)
	}
	if updated.Name != "Travel" {
		t.Fatalf("expected renamed folder Travel, got %q", updated.Name)
	}

	folders, err := svc.ListNoteFolders(ctx, user.ID)
	if err != nil {
		t.Fatalf("list note folders: %v", err)
	}
	if len(folders) != 1 || folders[0].Name != "Travel" {
		t.Fatalf("expected one Travel folder, got %+v", folders)
	}

	if err := svc.DeleteNoteFolder(ctx, user.ID, folder.ID); err != nil {
		t.Fatalf("delete note folder: %v", err)
	}

	reloaded, err := store.GetNoteByID(ctx, note.ID)
	if err != nil {
		t.Fatalf("reload note after folder delete: %v", err)
	}
	if reloaded.FolderID != nil {
		t.Fatalf("expected note folder_id cleared after folder delete, got %v", *reloaded.FolderID)
	}
}

func TestRecordServiceMoveNoteAndUpdateVisibility(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "note-move-user", Nickname: "note-move-user", Phone: "13800138016", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "note-move-group", InviteCode: "NOTE17", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	folder := &models.NoteFolder{GroupID: group.ID, OwnerID: user.ID, Name: "Pinned"}
	if err := store.CreateNoteFolder(ctx, folder); err != nil {
		t.Fatalf("create note folder: %v", err)
	}
	note := &models.Note{GroupID: group.ID, AuthorID: user.ID, Content: "move me", Type: "normal", Visibility: "private"}
	if err := store.CreateNote(ctx, note); err != nil {
		t.Fatalf("create note: %v", err)
	}

	moved, err := svc.MoveNoteToFolder(ctx, user.ID, note.ID, &folder.ID)
	if err != nil {
		t.Fatalf("move note to folder: %v", err)
	}
	if moved.FolderID == nil || *moved.FolderID != folder.ID {
		t.Fatalf("expected folder id %d, got %+v", folder.ID, moved.FolderID)
	}

	visible, err := svc.UpdateNoteVisibility(ctx, user.ID, note.ID, "public")
	if err != nil {
		t.Fatalf("update note visibility: %v", err)
	}
	if visible.Visibility != "public" {
		t.Fatalf("expected public visibility, got %q", visible.Visibility)
	}

	reset, err := svc.MoveNoteToFolder(ctx, user.ID, note.ID, nil)
	if err != nil {
		t.Fatalf("move note back to all: %v", err)
	}
	if reset.FolderID != nil {
		t.Fatalf("expected folder id nil after reset, got %+v", reset.FolderID)
	}
}

func TestRecordServiceListNotesWithFolderFilters(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "note-filter-owner", Nickname: "note-filter-owner", Phone: "13800138014", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	viewer := &models.User{WeChatID: "note-filter-viewer", Nickname: "note-filter-viewer", Phone: "13800138015", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, viewer); err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	group := &models.Group{Name: "note-filter-group", InviteCode: "NOTE15", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, viewer, group.ID); err != nil {
		t.Fatalf("add viewer to group: %v", err)
	}
	folder, err := svc.CreateNoteFolder(ctx, owner.ID, "Travel")
	if err != nil {
		t.Fatalf("create note folder: %v", err)
	}
	if _, err := svc.CreateNote(ctx, owner.ID, nil, "uncategorized", "#fff", "normal", nil, "public"); err != nil {
		t.Fatalf("create uncategorized note: %v", err)
	}
	if _, err := svc.CreateNote(ctx, owner.ID, &folder.ID, "custom note", "#000", "normal", nil, "public"); err != nil {
		t.Fatalf("create folder note: %v", err)
	}
	if _, err := svc.CreateNote(ctx, viewer.ID, nil, "viewer share", "#111", "normal", nil, "public"); err != nil {
		t.Fatalf("create viewer shared note: %v", err)
	}

	allResult, err := svc.ListNotes(ctx, owner.ID, NewPagination(1, 10), NoteListFilter{FolderType: "all"})
	if err != nil {
		t.Fatalf("list all notes: %v", err)
	}
	if len(allResult.Items) != 2 {
		t.Fatalf("expected both owned notes in all filter, got %+v", allResult.Items)
	}
	contents := map[string]bool{}
	for _, item := range allResult.Items {
		contents[item.Content] = true
	}
	if !contents["uncategorized"] || !contents["custom note"] {
		t.Fatalf("expected all filter to include categorized and uncategorized notes, got %+v", allResult.Items)
	}

	customResult, err := svc.ListNotes(ctx, owner.ID, NewPagination(1, 10), NoteListFilter{FolderType: "custom"})
	if err != nil {
		t.Fatalf("list custom notes: %v", err)
	}
	if len(customResult.Items) != 1 || customResult.Items[0].Content != "custom note" {
		t.Fatalf("expected custom note in custom filter, got %+v", customResult.Items)
	}

	folderResult, err := svc.ListNotes(ctx, owner.ID, NewPagination(1, 10), NoteListFilter{FolderID: &folder.ID})
	if err != nil {
		t.Fatalf("list notes by folder id: %v", err)
	}
	if len(folderResult.Items) != 1 || folderResult.Items[0].Content != "custom note" {
		t.Fatalf("expected custom note for folder id filter, got %+v", folderResult.Items)
	}

	sharedResult, err := svc.ListNotes(ctx, owner.ID, NewPagination(1, 10), NoteListFilter{FolderType: "shared"})
	if err != nil {
		t.Fatalf("list shared notes: %v", err)
	}
	if len(sharedResult.Items) != 1 || sharedResult.Items[0].Content != "viewer share" {
		t.Fatalf("expected viewer shared note in shared filter, got %+v", sharedResult.Items)
	}
}

func TestRecordServiceWishlistVisibilityCreateUpdateAndList(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "wishlist-owner", Nickname: "wishlist-owner", Phone: "13800138009", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	viewer := &models.User{WeChatID: "wishlist-viewer", Nickname: "wishlist-viewer", Phone: "13800138010", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, viewer); err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	group := &models.Group{Name: "wishlist-visibility-group", InviteCode: "WISH04", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, viewer, group.ID); err != nil {
		t.Fatalf("add viewer to group: %v", err)
	}

	item, err := svc.CreateWishlist(ctx, owner.ID, "private wish", "high", "private")
	if err != nil {
		t.Fatalf("create wishlist: %v", err)
	}
	if item.Visibility != "private" {
		t.Fatalf("expected private visibility, got %q", item.Visibility)
	}
	if item.CreatorID != owner.ID {
		t.Fatalf("expected creator id %d, got %d", owner.ID, item.CreatorID)
	}

	result, err := svc.ListWishlist(ctx, viewer.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list wishlist for viewer: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected private wishlist item to be hidden, got %d items", len(result.Items))
	}

	updated, err := svc.UpdateWishlist(ctx, owner.ID, item.ID, "public wish", "low", "public")
	if err != nil {
		t.Fatalf("update wishlist: %v", err)
	}
	if updated.Visibility != "public" {
		t.Fatalf("expected public visibility after update, got %q", updated.Visibility)
	}

	result, err = svc.ListWishlist(ctx, viewer.ID, NewPagination(1, 10))
	if err != nil {
		t.Fatalf("list wishlist after update: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 visible wishlist item after update, got %d", len(result.Items))
	}
}

func TestRecordServiceDeleteAlbum(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "delete-photo-user", Nickname: "delete-photo-user", Phone: "13800138011", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "delete-photo-group", InviteCode: "PHOTO3", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	album := &models.Album{
		GroupID:    group.ID,
		CreatorID:  user.ID,
		Title:      "delete album",
		Visibility: "public",
		Photos: []models.Photo{{
			GroupID:    group.ID,
			UploaderID: user.ID,
			URL:        "https://example.com/delete-photo.jpg",
		}},
	}
	if err := store.CreateAlbum(ctx, album); err != nil {
		t.Fatalf("create album: %v", err)
	}

	if err := svc.DeleteAlbum(ctx, user.ID, album.ID); err != nil {
		t.Fatalf("delete album: %v", err)
	}

	_, err := store.GetAlbumByID(ctx, album.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted album to be missing, got %v", err)
	}
}

func TestRecordServiceDeleteWishlist(t *testing.T) {
	store := newTestStore(t)
	svc := NewRecordService(store, newTestLogger())
	ctx := context.Background()

	user := &models.User{WeChatID: "delete-wishlist-user", Nickname: "delete-wishlist-user", Phone: "13800138013", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	group := &models.Group{Name: "delete-wishlist-group", InviteCode: "WISH05", CreatorID: user.ID}
	if err := store.CreateGroupWithCreator(ctx, user, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	item := &models.Wishlist{GroupID: group.ID, CreatorID: user.ID, Content: "delete wish"}
	if err := store.CreateWishlistItem(ctx, item); err != nil {
		t.Fatalf("create wishlist item: %v", err)
	}

	if err := svc.DeleteWishlist(ctx, user.ID, item.ID); err != nil {
		t.Fatalf("delete wishlist item: %v", err)
	}

	_, err := store.GetWishlistItemByID(ctx, item.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted wishlist item to be missing, got %v", err)
	}
}
