package handlers

import (
	"errors"
	"net/http"
	"time"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	existingUser, err := database.Queries.GetUserByWeChatID(req.WeChatID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user"})
			return
		}

		// Create new user
		user = models.User{
			WeChatID:        req.WeChatID,
			Nickname:        req.Nickname,
			AvatarURL:       req.AvatarURL,
			StatusUpdatedAt: time.Now(),
		}
		if err := database.Queries.CreateUser(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update info
		user = *existingUser
		user.Nickname = req.Nickname
		user.AvatarURL = req.AvatarURL
		if err := database.Queries.SaveUser(&user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
	}

	c.JSON(http.StatusOK, user)
}
