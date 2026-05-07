package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	GroupRequestTypeJoin              = "join"
	GroupRequestTypeOwnershipTransfer = "ownership_transfer"

	GroupRequestStatusPending  = "pending"
	GroupRequestStatusApproved = "approved"
	GroupRequestStatusRejected = "rejected"
)

type GroupRequest struct {
	gorm.Model
	GroupID         uint       `gorm:"index" json:"group_id"`
	RequesterUserID uint       `gorm:"index" json:"requester_user_id"`
	TargetUserID    uint       `gorm:"index" json:"target_user_id"`
	Type            string     `gorm:"size:40;index" json:"type"`
	Status          string     `gorm:"size:20;index" json:"status"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewerUserID  *uint      `gorm:"index" json:"reviewer_user_id,omitempty"`
}
