package models

import (
	"time"

	"gorm.io/gorm"
)

type Anniversary struct {
	gorm.Model
	UserID  uint      `gorm:"index" json:"user_id"`
	GroupID uint      `gorm:"index" json:"group_id"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
}
