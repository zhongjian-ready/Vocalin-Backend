package handlers

import (
	"net/http"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func currentUserID(c *gin.Context) uint {
	return c.MustGet("userID").(uint)
}

func mustCurrentUser(c *gin.Context) (*models.User, bool) {
	user, err := database.Queries.GetUserByID(currentUserID(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User not found"})
		return nil, false
	}

	return user, true
}

func mustCurrentGroupUser(c *gin.Context) (*models.User, uint, bool) {
	user, ok := mustCurrentUser(c)
	if !ok {
		return nil, 0, false
	}

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return nil, 0, false
	}

	return user, *user.GroupID, true
}