package handlers

import (
	"errors"
	"net/http"
	"time"
	"vocalin-backend/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateTimerRequest struct {
	Title     string    `json:"title" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdatePinnedMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// UpdateTimer godoc
// @Summary Update companion timer
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdateTimerRequest true "Update Timer Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Group
// @Router /home/timer [put]
func UpdateTimer(c *gin.Context) {
	var req UpdateTimerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	group, err := database.Queries.UpdateGroupTimer(groupID, req.Title, req.StartDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update timer"})
		return
	}

	c.JSON(http.StatusOK, *group)
}

// UpdateStatus godoc
// @Summary Update user status
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdateStatusRequest true "Update Status Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.User
// @Router /home/status [put]
func UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, ok := mustCurrentUser(c)
	if !ok {
		return
	}

	if err := database.Queries.UpdateUserStatus(user, req.Status, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, *user)
}

// UpdatePinnedMessage godoc
// @Summary Update pinned message
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdatePinnedMessageRequest true "Update Pinned Message Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Group
// @Router /home/pinned [put]
func UpdatePinnedMessage(c *gin.Context) {
	var req UpdatePinnedMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	group, err := database.Queries.UpdatePinnedMessage(groupID, user.ID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pinned message"})
		return
	}

	c.JSON(http.StatusOK, *group)
}

// GetHomeDashboard godoc
// @Summary Get home dashboard data
// @Tags Home
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /home/dashboard [get]
func GetHomeDashboard(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	group, err := database.Queries.GetGroupWithMembers(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Get recent activity (latest photo or note)
	latestPhoto, photoErr := database.Queries.GetLatestPhotoByGroup(group.ID)
	if photoErr != nil && !errors.Is(photoErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load recent photo"})
		return
	}

	latestNote, noteErr := database.Queries.GetLatestNoteByGroup(group.ID)
	if noteErr != nil && !errors.Is(noteErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load recent note"})
		return
	}

	var recentActivity interface{}
	if latestPhoto != nil && (latestNote == nil || latestPhoto.CreatedAt.After(latestNote.CreatedAt)) {
		recentActivity = map[string]interface{}{
			"type": "photo",
			"data": latestPhoto,
		}
	} else if latestNote != nil {
		recentActivity = map[string]interface{}{
			"type": "note",
			"data": latestNote,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"group":           group,
		"recent_activity": recentActivity,
	})
}
