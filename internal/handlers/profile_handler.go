package handlers

import (
	"net/http"
	"time"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateAnniversaryRequest struct {
	Title string    `json:"title" binding:"required"`
	Date  time.Time `json:"date" binding:"required"`
}

// CreateAnniversary godoc
// @Summary Add anniversary
// @Tags Profile
// @Accept json
// @Produce json
// @Param request body CreateAnniversaryRequest true "Create Anniversary Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Anniversary
// @Router /profile/anniversaries [post]
func CreateAnniversary(c *gin.Context) {
	var req CreateAnniversaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	anniversary := models.Anniversary{
		UserID:  user.ID,
		GroupID: groupID,
		Title:   req.Title,
		Date:    req.Date,
	}

	if err := database.Queries.CreateAnniversary(&anniversary); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create anniversary"})
		return
	}

	c.JSON(http.StatusOK, anniversary)
}

// GetAnniversaries godoc
// @Summary List anniversaries
// @Tags Profile
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.Anniversary
// @Router /profile/anniversaries [get]
func GetAnniversaries(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	anniversaries, err := database.Queries.ListAnniversariesByGroup(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load anniversaries"})
		return
	}

	c.JSON(http.StatusOK, anniversaries)
}

// LeaveGroup godoc
// @Summary Leave current group
// @Tags Profile
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string
// @Router /profile/leave [post]
func LeaveGroup(c *gin.Context) {
	user, _, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	if err := database.Queries.RemoveUserFromGroup(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

// ExportData godoc
// @Summary Export data to email
// @Tags Profile
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string
// @Router /profile/export [post]
func ExportData(c *gin.Context) {
	// Mock implementation
	c.JSON(http.StatusOK, gin.H{"message": "Data export started. You will receive an email shortly."})
}
