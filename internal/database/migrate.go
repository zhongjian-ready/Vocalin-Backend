package database

import (
	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

func ManagedModels() []any {
	return []any{
		&models.Group{},
		&models.User{},
		&models.GroupMember{},
		&models.Photo{},
		&models.Comment{},
		&models.Like{},
		&models.Note{},
		&models.Wishlist{},
		&models.Anniversary{},
		&models.RefreshToken{},
	}
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(ManagedModels()...); err != nil {
		return err
	}
	return backfillGroupMembers(db)
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
