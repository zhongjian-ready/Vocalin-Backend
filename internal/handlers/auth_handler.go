package handlers

import (
	"net/http"
	"time"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	WeChatID  string `json:"wechat_id" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// Login godoc
// @Summary User Login
// @Description Login with WeChat ID, creates user if not exists
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Request"
// @Success 200 {object} models.User
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	result := database.DB.Where("wechat_id = ?", req.WeChatID).First(&user)

	if result.Error != nil {
		// Create new user
		user = models.User{
			WeChatID:        req.WeChatID,
			Nickname:        req.Nickname,
			AvatarURL:       req.AvatarURL,
			StatusUpdatedAt: time.Now(),
		}
		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update info
		user.Nickname = req.Nickname
		user.AvatarURL = req.AvatarURL
		database.DB.Save(&user)
	}

	c.JSON(http.StatusOK, user)
}
