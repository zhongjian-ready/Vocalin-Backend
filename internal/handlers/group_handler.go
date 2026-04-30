package handlers

import (
	"net/http"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
	"vocalin-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// CreateGroup godoc
// @Summary Create a new group
// @Description Create a new group and join it
// @Tags Group
// @Accept json
// @Produce json
// @Param request body CreateGroupRequest true "Create Group Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Group
// @Router /groups [post]
func CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, ok := mustCurrentUser(c)
	if !ok {
		return
	}

	// Check if user is already in a group
	if user.GroupID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already in a group"})
		return
	}

	inviteCode := utils.GenerateInviteCode(6)
	group := models.Group{
		Name:       req.Name,
		InviteCode: inviteCode,
		CreatorID:  user.ID,
	}

	if err := database.Queries.CreateGroupWithCreator(user, &group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// JoinGroup godoc
// @Summary Join a group
// @Description Join a group by invite code
// @Tags Group
// @Accept json
// @Produce json
// @Param request body JoinGroupRequest true "Join Group Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Group
// @Router /groups/join [post]
func JoinGroup(c *gin.Context) {
	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, ok := mustCurrentUser(c)
	if !ok {
		return
	}

	if user.GroupID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already in a group"})
		return
	}

	group, err := database.Queries.GetGroupByInviteCode(req.InviteCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid invite code"})
		return
	}

	if err := database.Queries.AddUserToGroup(user, group.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join group"})
		return
	}

	c.JSON(http.StatusOK, *group)
}

// GetGroupInfo godoc
// @Summary Get current group info
// @Description Get info of the group the user belongs to
// @Tags Group
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.Group
// @Router /groups/me [get]
func GetGroupInfo(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	group, err := database.Queries.GetGroupWithMembers(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, *group)
}
