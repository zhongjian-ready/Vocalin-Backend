package models

import (
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	gorm.Model
	AlbumID    uint   `gorm:"index" json:"album_id"`
	GroupID    uint   `gorm:"index" json:"group_id"`
	UploaderID uint   `gorm:"column:uploader_id" json:"uploader_id"`
	URL        string `json:"url"`
}

type Album struct {
	gorm.Model
	GroupID     uint      `gorm:"index" json:"group_id"`
	CreatorID   uint      `gorm:"column:creator_id" json:"creator_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Visibility  string    `gorm:"size:20;default:public" json:"visibility"`
	Photos      []Photo   `gorm:"foreignKey:AlbumID" json:"photos"`
	Comments    []Comment `gorm:"foreignKey:AlbumID" json:"comments"`
	Likes       []Like    `gorm:"foreignKey:AlbumID" json:"likes"`
}

type Comment struct {
	gorm.Model
	AlbumID uint   `gorm:"column:album_id;index" json:"album_id"`
	UserID  uint   `json:"user_id"`
	Content string `json:"content"`
	User    User   `gorm:"foreignKey:UserID" json:"user"`
}

type Like struct {
	gorm.Model
	AlbumID uint `gorm:"column:album_id;index" json:"album_id"`
	UserID  uint `json:"user_id"`
}

type NoteFolder struct {
	gorm.Model
	GroupID uint   `gorm:"index" json:"group_id"`
	OwnerID uint   `gorm:"index" json:"owner_id"`
	Name    string `gorm:"size:100" json:"name"`
}

type Note struct {
	gorm.Model
	GroupID    uint        `gorm:"index" json:"group_id"`
	AuthorID   uint        `json:"author_id"`
	FolderID   *uint       `gorm:"index" json:"folder_id"`
	Content    string      `gorm:"type:longtext" json:"content"`
	Color      string      `json:"color"`
	Type       string      `json:"type"` // "normal", "burn", "timed"
	ShowAt     *time.Time  `json:"show_at"`
	Visibility string      `gorm:"size:20;default:public" json:"visibility"`
	IsBurned   bool        `json:"is_burned"`
	Author     User        `gorm:"foreignKey:AuthorID" json:"author"`
	Folder     *NoteFolder `gorm:"foreignKey:FolderID" json:"folder,omitempty"`
}

type Wishlist struct {
	gorm.Model
	GroupID     uint       `gorm:"index" json:"group_id"`
	CreatorID   uint       `json:"creator_id"`
	Content     string     `json:"content"`
	Priority    string     `gorm:"size:20;default:medium" json:"priority"`
	Visibility  string     `gorm:"size:20;default:public" json:"visibility"`
	IsCompleted bool       `json:"is_completed"`
	CompletedAt *time.Time `json:"completed_at"`
}
