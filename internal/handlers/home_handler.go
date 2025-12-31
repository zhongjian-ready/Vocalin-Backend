package handlers

import (
	"net/http"
	"time"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var group models.Group
	database.DB.First(&group, *user.GroupID)
	group.TimerTitle = req.Title
	group.TimerStartDate = req.StartDate
	database.DB.Save(&group)

	c.JSON(http.StatusOK, group)
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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	user.CurrentStatus = req.Status
	user.StatusUpdatedAt = time.Now()
	database.DB.Save(&user)

	c.JSON(http.StatusOK, user)
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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var group models.Group
	database.DB.First(&group, *user.GroupID)
	group.PinnedMessage = req.Content
	group.PinnedMessageAuthorID = userID
	database.DB.Save(&group)

	c.JSON(http.StatusOK, group)
}

// GetHomeDashboard godoc
// @Summary Get home dashboard data
// @Tags Home
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /home/dashboard [get]
func GetHomeDashboard(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var group models.Group
	database.DB.Preload("Members").First(&group, *user.GroupID)

	// Get recent activity (latest photo or note)
	var latestPhoto models.Photo
	database.DB.Where("group_id = ?", group.ID).Order("created_at desc").First(&latestPhoto)

	var latestNote models.Note
	database.DB.Where("group_id = ?", group.ID).Order("created_at desc").First(&latestNote)

	var recentActivity interface{}
	if latestPhoto.ID != 0 && (latestNote.ID == 0 || latestPhoto.CreatedAt.After(latestNote.CreatedAt)) {
		recentActivity = map[string]interface{}{
			"type": "photo",
			"data": latestPhoto,
		}
	} else if latestNote.ID != 0 {
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
