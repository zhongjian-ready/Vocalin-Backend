package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

func ManagedModels() []any {
	return []any{
		&models.Group{},
		&models.User{},
		&models.GroupMember{},
		&models.GroupRequest{},
		&models.Album{},
		&models.Photo{},
		&models.Comment{},
		&models.Like{},
		&models.NoteFolder{},
		&models.Note{},
		&models.Wishlist{},
		&models.Anniversary{},
		&models.RefreshToken{},
	}
}

func AutoMigrate(db *gorm.DB) error {
	legacyPhotos, err := snapshotLegacyPhotoAlbums(db)
	if err != nil {
		return err
	}
	legacyCommentLinks, err := snapshotLegacyAlbumReactionLinks(db, "comments")
	if err != nil {
		return err
	}
	legacyLikeLinks, err := snapshotLegacyAlbumReactionLinks(db, "likes")
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(ManagedModels()...); err != nil {
		return err
	}
	if err := backfillLegacyPhotoAlbums(db, legacyPhotos); err != nil {
		return err
	}
	if err := dropLegacyPhotoColumns(db); err != nil {
		return err
	}
	if err := backfillLegacyAlbumReactions(db, "comments", &legacyCommentMigrationModel{}, legacyCommentLinks); err != nil {
		return err
	}
	if err := backfillLegacyAlbumReactions(db, "likes", &legacyLikeMigrationModel{}, legacyLikeLinks); err != nil {
		return err
	}
	if err := backfillRecordVisibility(db); err != nil {
		return err
	}
	return backfillGroupMembers(db)
}

type legacyPhotoAlbumRow struct {
	ID          int       `gorm:"column:id"`
	GroupID     int       `gorm:"column:group_id"`
	UploaderID  int       `gorm:"column:uploader_id"`
	Description string    `gorm:"column:description"`
	Visibility  string    `gorm:"column:visibility"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type legacyAlbumReactionLink struct {
	ID      int
	PhotoID int
}

type legacyPhotoColumnsMigrationModel struct {
	models.Photo
	Description string `gorm:"column:description"`
	Source      string `gorm:"column:source"`
}

func (legacyPhotoColumnsMigrationModel) TableName() string {
	return "photos"
}

type legacyCommentMigrationModel struct {
	models.Comment
	PhotoID uint `gorm:"column:photo_id"`
}

func (legacyCommentMigrationModel) TableName() string {
	return "comments"
}

type legacyLikeMigrationModel struct {
	models.Like
	PhotoID uint `gorm:"column:photo_id"`
}

func (legacyLikeMigrationModel) TableName() string {
	return "likes"
}

func snapshotLegacyPhotoAlbums(db *gorm.DB) ([]legacyPhotoAlbumRow, error) {
	if !db.Migrator().HasTable("photos") {
		return nil, nil
	}

	query := legacyPhotoSnapshotQuery(
		db.Migrator().HasColumn("photos", "description"),
		db.Migrator().HasColumn("photos", "visibility"),
		db.Migrator().HasColumn("photos", "album_id"),
	)
	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	legacyPhotos := make([]legacyPhotoAlbumRow, 0)
	for rows.Next() {
		var row legacyPhotoAlbumRow
		var description sql.NullString
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&row.ID, &row.GroupID, &row.UploaderID, &description, &createdAt, &updatedAt, &row.Visibility); err != nil {
			return nil, err
		}
		row.Description = description.String
		if createdAt.Valid {
			row.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			row.UpdatedAt = updatedAt.Time
		}
		legacyPhotos = append(legacyPhotos, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return legacyPhotos, nil
}

func backfillLegacyPhotoAlbums(db *gorm.DB, legacyPhotos []legacyPhotoAlbumRow) error {
	if !db.Migrator().HasTable(&models.Photo{}) || !db.Migrator().HasTable(&models.Album{}) || len(legacyPhotos) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, legacyPhoto := range legacyPhotos {
			var existingAlbumID sql.NullInt64
			if err := tx.Table("photos").Where("id = ?", legacyPhoto.ID).Select("album_id").Scan(&existingAlbumID).Error; err != nil {
				return err
			}
			if existingAlbumID.Valid && existingAlbumID.Int64 != 0 {
				continue
			}
			album := models.Album{
				Model: gorm.Model{
					CreatedAt: legacyPhoto.CreatedAt,
					UpdatedAt: legacyPhoto.UpdatedAt,
				},
				GroupID:     uint(legacyPhoto.GroupID),
				CreatorID:   uint(legacyPhoto.UploaderID),
				Title:       buildLegacyAlbumTitle(legacyPhoto.Description, legacyPhoto.CreatedAt),
				Description: legacyPhoto.Description,
				Visibility:  normalizeLegacyRecordVisibility(legacyPhoto.Visibility),
			}
			if err := tx.Create(&album).Error; err != nil {
				return err
			}
			if err := tx.Table("photos").Where("id = ?", legacyPhoto.ID).Updates(map[string]any{
				"album_id":    album.ID,
				"group_id":    uint(legacyPhoto.GroupID),
				"uploader_id": uint(legacyPhoto.UploaderID),
				"description": legacyPhoto.Description,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func snapshotLegacyAlbumReactionLinks(db *gorm.DB, table string) ([]legacyAlbumReactionLink, error) {
	if !db.Migrator().HasTable(table) || !db.Migrator().HasColumn(table, "photo_id") {
		return nil, nil
	}

	rows, err := db.Raw(fmt.Sprintf("SELECT id, photo_id FROM %s WHERE deleted_at IS NULL", table)).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]legacyAlbumReactionLink, 0)
	for rows.Next() {
		var link legacyAlbumReactionLink
		if err := rows.Scan(&link.ID, &link.PhotoID); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

func backfillLegacyAlbumReactions(db *gorm.DB, table string, model any, links []legacyAlbumReactionLink) error {
	if !db.Migrator().HasTable(model) || !db.Migrator().HasColumn(model, "album_id") {
		return nil
	}

	if len(links) > 0 {
		photoAlbumIDs, err := loadPhotoAlbumIDs(db)
		if err != nil {
			return err
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, link := range links {
				albumID, ok := photoAlbumIDs[link.PhotoID]
				if !ok || albumID == 0 {
					continue
				}

				var existingAlbumID sql.NullInt64
				if err := tx.Table(table).Where("id = ?", link.ID).Select("album_id").Scan(&existingAlbumID).Error; err != nil {
					return err
				}
				if existingAlbumID.Valid && existingAlbumID.Int64 != 0 {
					continue
				}

				if err := tx.Table(table).Where("id = ?", link.ID).Update("album_id", albumID).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return dropLegacyReactionPhotoIDColumn(db, table, model)
}

func loadPhotoAlbumIDs(db *gorm.DB) (map[int]uint, error) {
	type photoAlbumPair struct {
		ID      int  `gorm:"column:id"`
		AlbumID uint `gorm:"column:album_id"`
	}

	var rows []photoAlbumPair
	if err := db.Table("photos").Select("id, album_id").Where("album_id IS NOT NULL AND album_id <> 0").Scan(&rows).Error; err != nil {
		return nil, err
	}
	photoAlbumIDs := make(map[int]uint, len(rows))
	for _, row := range rows {
		photoAlbumIDs[row.ID] = row.AlbumID
	}
	return photoAlbumIDs, nil
}

func dropLegacyPhotoColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Photo{}) {
		return nil
	}

	columnsToDrop := make([]string, 0, 2)
	for _, column := range []string{"description", "source"} {
		if db.Migrator().HasColumn("photos", column) {
			columnsToDrop = append(columnsToDrop, column)
		}
	}
	if len(columnsToDrop) == 0 {
		return nil
	}

	if db.Dialector.Name() != "sqlite" {
		for _, column := range columnsToDrop {
			if err := db.Migrator().DropColumn(&legacyPhotoColumnsMigrationModel{}, column); err != nil {
				return err
			}
		}
		return nil
	}

	return rebuildPhotosWithoutLegacyColumns(db)
}

func rebuildPhotosWithoutLegacyColumns(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DROP TABLE IF EXISTS photos__column_cleanup").Error; err != nil {
			return err
		}
		if err := tx.Exec(`CREATE TABLE photos__column_cleanup (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			album_id integer,
			group_id integer,
			uploader_id integer,
			url text
		)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO photos__column_cleanup (id, created_at, updated_at, deleted_at, album_id, group_id, uploader_id, url)
			SELECT id, created_at, updated_at, deleted_at, album_id, group_id, uploader_id, url FROM photos`).Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP TABLE photos").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE photos__column_cleanup RENAME TO photos").Error; err != nil {
			return err
		}
		for _, statement := range []string{
			"CREATE INDEX IF NOT EXISTS idx_photos_deleted_at ON photos(deleted_at)",
			"CREATE INDEX IF NOT EXISTS idx_photos_album_id ON photos(album_id)",
			"CREATE INDEX IF NOT EXISTS idx_photos_group_id ON photos(group_id)",
		} {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func dropLegacyReactionPhotoIDColumn(db *gorm.DB, table string, model any) error {
	if !db.Migrator().HasColumn(table, "photo_id") {
		return nil
	}
	if db.Dialector.Name() != "sqlite" {
		if err := dropForeignKeysForColumn(db, table, "photo_id"); err != nil {
			return err
		}
		return db.Migrator().DropColumn(model, "photo_id")
	}

	var createSQL string
	var copySQL string
	var indexSQL []string

	switch table {
	case "comments":
		createSQL = `CREATE TABLE comments__album_migration (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			album_id integer,
			user_id integer,
			content text
		)`
		copySQL = `INSERT INTO comments__album_migration (id, created_at, updated_at, deleted_at, album_id, user_id, content)
			SELECT id, created_at, updated_at, deleted_at, album_id, user_id, content FROM comments`
		indexSQL = []string{
			"CREATE INDEX IF NOT EXISTS idx_comments_deleted_at ON comments(deleted_at)",
			"CREATE INDEX IF NOT EXISTS idx_comments_album_id ON comments(album_id)",
		}
	case "likes":
		createSQL = `CREATE TABLE likes__album_migration (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			album_id integer,
			user_id integer
		)`
		copySQL = `INSERT INTO likes__album_migration (id, created_at, updated_at, deleted_at, album_id, user_id)
			SELECT id, created_at, updated_at, deleted_at, album_id, user_id FROM likes`
		indexSQL = []string{
			"CREATE INDEX IF NOT EXISTS idx_likes_deleted_at ON likes(deleted_at)",
			"CREATE INDEX IF NOT EXISTS idx_likes_album_id ON likes(album_id)",
		}
	default:
		return fmt.Errorf("unsupported reaction table: %s", table)
	}

	tempTable := table + "__album_migration"
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DROP TABLE IF EXISTS " + tempTable).Error; err != nil {
			return err
		}
		if err := tx.Exec(createSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec(copySQL).Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP TABLE " + table).Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE " + tempTable + " RENAME TO " + table).Error; err != nil {
			return err
		}
		for _, statement := range indexSQL {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func dropForeignKeysForColumn(db *gorm.DB, table string, column string) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}

	rows, err := db.Raw(
		`SELECT constraint_name
		 FROM information_schema.KEY_COLUMN_USAGE
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND column_name = ?
		   AND referenced_table_name IS NOT NULL`,
		table,
		column,
	).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var constraintName string
		if err := rows.Scan(&constraintName); err != nil {
			return err
		}
		if err := db.Exec(
			fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", table, constraintName),
		).Error; err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func legacyPhotoBackfillSelectClause(hasDescriptionColumn bool, hasVisibilityColumn bool) string {
	descriptionClause := "'' AS description"
	if hasDescriptionColumn {
		descriptionClause = "description"
	}
	if hasVisibilityColumn {
		return fmt.Sprintf("id, group_id, uploader_id, %s, created_at, updated_at, COALESCE(visibility, 'public') AS visibility", descriptionClause)
	}
	return fmt.Sprintf("id, group_id, uploader_id, %s, created_at, updated_at, 'public' AS visibility", descriptionClause)
}

func legacyPhotoSnapshotQuery(hasDescriptionColumn bool, hasVisibilityColumn bool, hasAlbumIDColumn bool) string {
	whereClause := "deleted_at IS NULL"
	if hasAlbumIDColumn {
		whereClause += " AND (album_id = 0 OR album_id IS NULL)"
	}
	return fmt.Sprintf("SELECT %s FROM photos WHERE %s ORDER BY created_at asc, id asc", legacyPhotoBackfillSelectClause(hasDescriptionColumn, hasVisibilityColumn), whereClause)
}

func buildLegacyAlbumTitle(description string, createdAt time.Time) string {
	trimmed := strings.TrimSpace(description)
	if trimmed != "" {
		runes := []rune(trimmed)
		if len(runes) > 255 {
			return string(runes[:255])
		}
		return trimmed
	}
	return fmt.Sprintf("历史相册 %s", createdAt.Format("2006-01-02"))
}

func normalizeLegacyRecordVisibility(visibility string) string {
	trimmed := strings.TrimSpace(visibility)
	if trimmed == "private" {
		return "private"
	}
	return "public"
}

func backfillRecordVisibility(db *gorm.DB) error {
	updates := []struct {
		model any
	}{
		{model: &models.Album{}},
		{model: &models.Note{}},
		{model: &models.Wishlist{}},
	}

	for _, update := range updates {
		if err := db.Model(update.model).Where("visibility = '' OR visibility IS NULL").Update("visibility", "public").Error; err != nil {
			return err
		}
	}

	return nil
}

func backfillGroupMembers(db *gorm.DB) error {
	var users []models.User
	if err := db.Where("group_id IS NOT NULL").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if user.CurrentGroupID == nil {
			continue
		}
		role := "member"
		var group models.Group
		if err := db.First(&group, *user.CurrentGroupID).Error; err == nil && group.CreatorID == user.ID {
			role = "owner"
		}
		membership := models.GroupMember{UserID: user.ID, GroupID: *user.CurrentGroupID}
		if err := db.Where(models.GroupMember{UserID: user.ID, GroupID: *user.CurrentGroupID}).Attrs(models.GroupMember{Role: role}).FirstOrCreate(&membership).Error; err != nil {
			return err
		}
		if membership.Role == "" {
			membership.Role = role
			if err := db.Save(&membership).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func HasColumn(db *gorm.DB, value any, column string) bool {
	return db.Migrator().HasColumn(value, column)
}

func DropColumn(db *gorm.DB, value any, column string) error {
	return db.Migrator().DropColumn(value, column)
}
