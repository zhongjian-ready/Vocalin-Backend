package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	WeChatID        string    `gorm:"column:wechat_id;uniqueIndex;size:100" json:"wechat_id"`
	Nickname        string    `gorm:"size:50;index" json:"nickname"`
	Phone           string    `gorm:"size:20;index" json:"phone,omitempty"`
	PasswordHash    string    `gorm:"size:255" json:"-"`
	AvatarURL       string    `json:"avatar_url"`
	CurrentGroupID  *uint     `gorm:"column:group_id" json:"current_group_id"`
	CurrentStatus   string    `json:"current_status"`
	StatusUpdatedAt time.Time `json:"status_updated_at"`
	GroupRole       string    `gorm:"column:group_role;->;-:migration" json:"role,omitempty"`
}

type Group struct {
	gorm.Model
	Name                  string     `json:"name"`
	InviteCode            string     `gorm:"uniqueIndex;size:20" json:"invite_code"`
	CreatorID             uint       `json:"creator_id"`
	Members               []User     `gorm:"-" json:"members"`
	TimerTitle            string     `json:"timer_title"`
	TimerStartDate        *time.Time `json:"timer_start_date"`
	PinnedMessage         string     `json:"pinned_message"`
	PinnedMessageAuthorID uint       `json:"pinned_message_author_id"`
	MyRole                string     `gorm:"-" json:"my_role,omitempty"`
}

type GroupMember struct {
	gorm.Model
	UserID  uint   `gorm:"uniqueIndex:idx_group_members_user_group;index" json:"user_id"`
	GroupID uint   `gorm:"uniqueIndex:idx_group_members_user_group;index" json:"group_id"`
	Role    string `gorm:"size:20" json:"role"`
}

// RefreshToken 用于维护可撤销的刷新令牌会话。
type RefreshToken struct {
	gorm.Model
	TokenID           string     `gorm:"uniqueIndex;size:64" json:"token_id"`
	UserID            uint       `gorm:"index" json:"user_id"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at"`
	ReplacedByTokenID string     `gorm:"size:64" json:"replaced_by_token_id"`
}
