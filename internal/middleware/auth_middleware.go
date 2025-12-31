package middleware

import (
	"net/http"
	"strconv"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: X-User-ID header missing"})
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid User ID"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User not found"})
			c.Abort()
			return
		}

		c.Set("userID", uint(userID))
		c.Set("user", user)
		c.Next()
	}
}
