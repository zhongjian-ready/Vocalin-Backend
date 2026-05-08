package database

import (
	"context"
	"testing"
	"time"

	"vocalin-backend/internal/models"
	"vocalin-backend/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrateBackfillsLegacyPhotosIntoAlbums(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE photos (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			group_id integer,
			uploader_id integer,
			url text,
			description text,
			source text,
			visibility text
		)
	`).Error; err != nil {
		t.Fatalf("create legacy photos table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE comments (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			photo_id integer,
			user_id integer,
			content text
		)
	`).Error; err != nil {
		t.Fatalf("create legacy comments table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE likes (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			photo_id integer,
			user_id integer
		)
	`).Error; err != nil {
		t.Fatalf("create legacy likes table: %v", err)
	}

	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	if err := db.Exec(
		"INSERT INTO photos (id, created_at, updated_at, group_id, uploader_id, url, description, visibility) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		1,
		createdAt,
		createdAt,
		7,
		11,
		"https://example.com/legacy.jpg",
		"旧照片记录",
		"private",
	).Error; err != nil {
		t.Fatalf("insert legacy photo: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO comments (id, created_at, updated_at, photo_id, user_id, content) VALUES (?, ?, ?, ?, ?, ?)",
		1,
		createdAt,
		createdAt,
		1,
		21,
		"legacy comment",
	).Error; err != nil {
		t.Fatalf("insert legacy comment: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO likes (id, created_at, updated_at, photo_id, user_id) VALUES (?, ?, ?, ?, ?)",
		1,
		createdAt,
		createdAt,
		1,
		22,
	).Error; err != nil {
		t.Fatalf("insert legacy like: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	var rawLegacyUploaderID int
	if err := db.Table("photos").Select("uploader_id").Where("id = ?", 1).Scan(&rawLegacyUploaderID).Error; err != nil {
		t.Fatalf("read legacy uploader_id after migrate: %v", err)
	}
	if rawLegacyUploaderID != 11 {
		t.Fatalf("expected legacy uploader_id 11 after migrate, got %d", rawLegacyUploaderID)
	}

	store := repository.NewStore(db)
	albums, total, err := store.ListAlbumsByGroup(context.Background(), 7, 11, 0, 20)
	if err != nil {
		t.Fatalf("list albums by group: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("expected 1 backfilled album, got %d", len(albums))
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if albums[0].GroupID != 7 {
		t.Fatalf("expected group_id 7, got %d", albums[0].GroupID)
	}
	if albums[0].Visibility != "private" {
		t.Fatalf("expected private visibility, got %q", albums[0].Visibility)
	}
	if albums[0].Title != "旧照片记录" {
		t.Fatalf("expected title derived from description, got %q", albums[0].Title)
	}
	if len(albums[0].Photos) != 1 {
		t.Fatalf("expected 1 nested photo, got %d", len(albums[0].Photos))
	}
	if HasColumn(db, "photos", "description") {
		t.Fatal("expected legacy photos.description column to be dropped")
	}
	if HasColumn(db, "photos", "source") {
		t.Fatal("expected legacy photos.source column to be dropped")
	}

	var rawCreatorID int
	if err := db.Table("albums").Select("creator_id").Where("id = ?", albums[0].ID).Scan(&rawCreatorID).Error; err != nil {
		t.Fatalf("read raw creator_id: %v", err)
	}
	if rawCreatorID != 11 {
		t.Fatalf("expected raw creator_id 11, got %d", rawCreatorID)
	}

	var rawCommentAlbumID int
	if err := db.Table("comments").Select("album_id").Where("id = ?", 1).Scan(&rawCommentAlbumID).Error; err != nil {
		t.Fatalf("read comment album_id: %v", err)
	}
	if rawCommentAlbumID != int(albums[0].ID) {
		t.Fatalf("expected comment album_id %d, got %d", albums[0].ID, rawCommentAlbumID)
	}
	if HasColumn(db, "comments", "photo_id") {
		t.Fatal("expected legacy comments.photo_id column to be dropped")
	}

	var rawLikeAlbumID int
	if err := db.Table("likes").Select("album_id").Where("id = ?", 1).Scan(&rawLikeAlbumID).Error; err != nil {
		t.Fatalf("read like album_id: %v", err)
	}
	if rawLikeAlbumID != int(albums[0].ID) {
		t.Fatalf("expected like album_id %d, got %d", albums[0].ID, rawLikeAlbumID)
	}
	if HasColumn(db, "likes", "photo_id") {
		t.Fatal("expected legacy likes.photo_id column to be dropped")
	}

	var migratedPhoto models.Photo
	if err := db.First(&migratedPhoto, 1).Error; err != nil {
		t.Fatalf("reload migrated photo: %v", err)
	}
	var rawUploaderID int
	if err := db.Table("photos").Select("uploader_id").Where("id = ?", migratedPhoto.ID).Scan(&rawUploaderID).Error; err != nil {
		t.Fatalf("read raw uploader_id: %v", err)
	}
	if rawUploaderID != 11 {
		t.Fatalf("expected raw uploader_id 11, got %d", rawUploaderID)
	}
	if migratedPhoto.AlbumID != albums[0].ID {
		t.Fatalf("expected stored photo album_id %d, got %d", albums[0].ID, migratedPhoto.AlbumID)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}

	var albumCount int64
	if err := db.Model(&models.Album{}).Count(&albumCount).Error; err != nil {
		t.Fatalf("count albums: %v", err)
	}
	if albumCount != 1 {
		t.Fatalf("expected idempotent migration, got %d albums", albumCount)
	}
}
