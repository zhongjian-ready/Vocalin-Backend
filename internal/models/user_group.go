package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	WeChatID        string    `gorm:"column:wechat_id;uniqueIndex;size:100" json:"wechat_id"`
	Nickname        string    `json:"nickname"`
	AvatarURL       string    `json:"avatar_url"`
	GroupID         *uint     `json:"group_id"` // Nullable if not in a group
	CurrentStatus   string    `json:"current_status"`
	StatusUpdatedAt time.Time `json:"status_updated_at"`
}

type Group struct {
	gorm.Model
	Name                  string    `json:"name"`
	InviteCode            string    `gorm:"uniqueIndex;size:20" json:"invite_code"`
	CreatorID             uint      `json:"creator_id"`
	Members               []User    `gorm:"foreignKey:GroupID" json:"members"`
	TimerTitle            string    `json:"timer_title"`
	TimerStartDate        time.Time `json:"timer_start_date"`
	PinnedMessage         string    `json:"pinned_message"`
	PinnedMessageAuthorID uint      `json:"pinned_message_author_id"`
}
