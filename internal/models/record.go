package models

import (
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	gorm.Model
	GroupID     uint      `gorm:"index" json:"group_id"`
	UploaderID  uint      `json:"uploader_id"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Comments    []Comment `gorm:"foreignKey:PhotoID" json:"comments"`
	Likes       []Like    `gorm:"foreignKey:PhotoID" json:"likes"`
}

type Comment struct {
	gorm.Model
	PhotoID uint   `gorm:"index" json:"photo_id"`
	UserID  uint   `json:"user_id"`
	Content string `json:"content"`
	User    User   `gorm:"foreignKey:UserID" json:"user"`
}

type Like struct {
	gorm.Model
	PhotoID uint `gorm:"index" json:"photo_id"`
	UserID  uint `json:"user_id"`
}

type Note struct {
	gorm.Model
	GroupID  uint       `gorm:"index" json:"group_id"`
	AuthorID uint       `json:"author_id"`
	Content  string     `json:"content"`
	Color    string     `json:"color"`
	Type     string     `json:"type"` // "normal", "burn", "timed"
	ShowAt   *time.Time `json:"show_at"`
	IsBurned bool       `json:"is_burned"`
	Author   User       `gorm:"foreignKey:AuthorID" json:"author"`
}

type Wishlist struct {
	gorm.Model
	GroupID     uint       `gorm:"index" json:"group_id"`
	Content     string     `json:"content"`
	IsCompleted bool       `json:"is_completed"`
	CompletedAt *time.Time `json:"completed_at"`
}
