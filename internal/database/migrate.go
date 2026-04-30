package database

import (
	"vocalin-backend/internal/models"

	"gorm.io/gorm"
)

func ManagedModels() []any {
	return []any{
		&models.Group{},
		&models.User{},
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
	return db.AutoMigrate(ManagedModels()...)
}

func HasColumn(db *gorm.DB, value any, column string) bool {
	return db.Migrator().HasColumn(value, column)
}

func DropColumn(db *gorm.DB, value any, column string) error {
	return db.Migrator().DropColumn(value, column)
}
