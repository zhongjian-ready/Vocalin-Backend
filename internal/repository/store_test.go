package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(database.ManagedModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewStore(db)
}

func TestCreateGroupWithCreatorOnlyUpdatesGroupID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	creator := &models.User{
		WeChatID:        "creator-wechat",
		Nickname:        "creator",
		Phone:           "13800138010",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, creator); err != nil {
		t.Fatalf("create creator: %v", err)
	}

	otherUser := &models.User{
		WeChatID:        "other-wechat",
		Nickname:        "other",
		Phone:           "13800138011",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, otherUser); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	creator.WeChatID = otherUser.WeChatID
	group := &models.Group{Name: "Warm Home", InviteCode: "WARM01", CreatorID: creator.ID}

	if err := store.CreateGroupWithCreator(ctx, creator, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	reloaded, err := store.GetUserByID(ctx, creator.ID)
	if err != nil {
		t.Fatalf("reload creator: %v", err)
	}
	if reloaded.CurrentGroupID == nil || *reloaded.CurrentGroupID != group.ID {
		t.Fatalf("expected creator current group id %d, got %v", group.ID, reloaded.CurrentGroupID)
	}
	if reloaded.WeChatID != "creator-wechat" {
		t.Fatalf("expected creator wechat id to remain unchanged, got %s", reloaded.WeChatID)
	}
	if creator.CurrentGroupID == nil || *creator.CurrentGroupID != group.ID {
		t.Fatalf("expected in-memory creator current group id %d, got %v", group.ID, creator.CurrentGroupID)
	}
	membership, err := store.GetGroupMember(ctx, group.ID, creator.ID)
	if err != nil {
		t.Fatalf("load creator membership: %v", err)
	}
	if membership.Role != groupRoleOwner {
		t.Fatalf("expected creator role %q, got %q", groupRoleOwner, membership.Role)
	}
}

func TestActiveGroupQueriesExcludeSoftDeletedMemberships(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	owner := &models.User{
		WeChatID:        "owner-soft-delete-wechat",
		Nickname:        "owner-soft-delete",
		Phone:           "13800138012",
		StatusUpdatedAt: time.Now(),
	}
	member := &models.User{
		WeChatID:        "member-soft-delete-wechat",
		Nickname:        "member-soft-delete",
		Phone:           "13800138013",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	group := &models.Group{Name: "Quiet Room", InviteCode: "QUIET1", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group.ID); err != nil {
		t.Fatalf("add member to group: %v", err)
	}
	if _, err := store.RemoveUserFromGroup(ctx, member.ID, group.ID); err != nil {
		t.Fatalf("remove member from group: %v", err)
	}

	loadedGroup, err := store.GetGroupWithMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("get group with members: %v", err)
	}
	if len(loadedGroup.Members) != 1 {
		t.Fatalf("expected 1 active member, got %d", len(loadedGroup.Members))
	}
	if loadedGroup.Members[0].ID != owner.ID {
		t.Fatalf("expected owner %d to remain in member list, got %+v", owner.ID, loadedGroup.Members)
	}

	groups, err := store.ListGroupsByUser(ctx, member.ID)
	if err != nil {
		t.Fatalf("list groups by removed user: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected removed user to have no active groups, got %+v", groups)
	}

	_, err = store.GetFirstGroupByUser(ctx, member.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestAddUserToGroupRestoresSoftDeletedMembership(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	owner := &models.User{
		WeChatID:        "owner-rejoin-wechat",
		Nickname:        "owner-rejoin",
		Phone:           "13800138014",
		StatusUpdatedAt: time.Now(),
	}
	member := &models.User{
		WeChatID:        "member-rejoin-wechat",
		Nickname:        "member-rejoin",
		Phone:           "13800138015",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	group := &models.Group{Name: "Rejoin Room", InviteCode: "REJOIN", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group.ID); err != nil {
		t.Fatalf("add member to group: %v", err)
	}
	if _, err := store.RemoveUserFromGroup(ctx, member.ID, group.ID); err != nil {
		t.Fatalf("remove member from group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group.ID); err != nil {
		t.Fatalf("re-add member to group: %v", err)
	}

	membership, err := store.GetGroupMember(ctx, group.ID, member.ID)
	if err != nil {
		t.Fatalf("get restored membership: %v", err)
	}
	if membership.Role != groupRoleMember {
		t.Fatalf("expected restored role %q, got %q", groupRoleMember, membership.Role)
	}

	var membershipCount int64
	if err := store.db.WithContext(ctx).Unscoped().Model(&models.GroupMember{}).Where("user_id = ? AND group_id = ?", member.ID, group.ID).Count(&membershipCount).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if membershipCount != 1 {
		t.Fatalf("expected one membership row after restore, got %d", membershipCount)
	}

	reloaded, err := store.GetUserByID(ctx, member.ID)
	if err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if reloaded.CurrentGroupID == nil || *reloaded.CurrentGroupID != group.ID {
		t.Fatalf("expected current group id %d, got %v", group.ID, reloaded.CurrentGroupID)
	}
}

func TestAlbumReactionsStayOnAlbumWhenReplacingPhotos(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	owner := &models.User{
		WeChatID:        "album-reaction-owner",
		Nickname:        "album-reaction-owner",
		Phone:           "13800138016",
		StatusUpdatedAt: time.Now(),
	}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}

	group := &models.Group{Name: "Album Reactions", InviteCode: "ALBREA", CreatorID: owner.ID}
	if err := store.CreateGroupWithCreator(ctx, owner, group); err != nil {
		t.Fatalf("create group with creator: %v", err)
	}

	album := &models.Album{
		GroupID:     group.ID,
		CreatorID:   owner.ID,
		Title:       "album",
		Description: "album",
		Visibility:  "public",
		Photos: []models.Photo{{
			GroupID:    group.ID,
			UploaderID: owner.ID,
			URL:        "https://example.com/original.jpg",
		}},
	}
	if err := store.CreateAlbum(ctx, album); err != nil {
		t.Fatalf("create album: %v", err)
	}
	if err := store.db.WithContext(ctx).Create(&models.Comment{AlbumID: album.ID, UserID: owner.ID, Content: "keep me"}).Error; err != nil {
		t.Fatalf("create album comment: %v", err)
	}
	if err := store.db.WithContext(ctx).Create(&models.Like{AlbumID: album.ID, UserID: owner.ID}).Error; err != nil {
		t.Fatalf("create album like: %v", err)
	}

	if err := store.ReplaceAlbumPhotos(ctx, album.ID, []models.Photo{{
		GroupID:    group.ID,
		UploaderID: owner.ID,
		URL:        "https://example.com/replaced.jpg",
	}}); err != nil {
		t.Fatalf("replace album photos: %v", err)
	}

	var commentCount int64
	if err := store.db.WithContext(ctx).Model(&models.Comment{}).Where("album_id = ?", album.ID).Count(&commentCount).Error; err != nil {
		t.Fatalf("count album comments: %v", err)
	}
	if commentCount != 1 {
		t.Fatalf("expected album comments to remain after photo replacement, got %d", commentCount)
	}

	var likeCount int64
	if err := store.db.WithContext(ctx).Model(&models.Like{}).Where("album_id = ?", album.ID).Count(&likeCount).Error; err != nil {
		t.Fatalf("count album likes: %v", err)
	}
	if likeCount != 1 {
		t.Fatalf("expected album likes to remain after photo replacement, got %d", likeCount)
	}

	if err := store.DeleteAlbum(ctx, album.ID); err != nil {
		t.Fatalf("delete album: %v", err)
	}

	if err := store.db.WithContext(ctx).Model(&models.Comment{}).Where("album_id = ?", album.ID).Count(&commentCount).Error; err != nil {
		t.Fatalf("count album comments after delete: %v", err)
	}
	if commentCount != 0 {
		t.Fatalf("expected album comments to be deleted with album, got %d", commentCount)
	}

	if err := store.db.WithContext(ctx).Model(&models.Like{}).Where("album_id = ?", album.ID).Count(&likeCount).Error; err != nil {
		t.Fatalf("count album likes after delete: %v", err)
	}
	if likeCount != 0 {
		t.Fatalf("expected album likes to be deleted with album, got %d", likeCount)
	}
}
