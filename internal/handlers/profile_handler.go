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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	anniversary := models.Anniversary{
		UserID:  userID,
		GroupID: *user.GroupID,
		Title:   req.Title,
		Date:    req.Date,
	}

	if err := database.DB.Create(&anniversary).Error; err != nil {
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
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var anniversaries []models.Anniversary
	database.DB.Where("group_id = ?", *user.GroupID).Order("date asc").Find(&anniversaries)

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
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	user.GroupID = nil
	database.DB.Save(&user)

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
