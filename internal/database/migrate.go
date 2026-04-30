package database

import "vocalin-backend/internal/models"

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
	}
}

func AutoMigrate() error {
	return DB.AutoMigrate(ManagedModels()...)
}

func HasColumn(value any, column string) bool {
	return DB.Migrator().HasColumn(value, column)
}

func DropColumn(value any, column string) error {
	return DB.Migrator().DropColumn(value, column)
}
