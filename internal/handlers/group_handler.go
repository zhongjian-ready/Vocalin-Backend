package handlers

import (
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	groupService *service.GroupService
}

type GroupResponse = models.Group

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required,min=2,max=50"`
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required,invite_code"`
}

func NewGroupHandler(groupService *service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// CreateGroup godoc
// @Summary 创建空间
// @Description 创建一个新的亲密空间，并自动将当前用户加入空间
// @Tags Group
// @Accept json
// @Produce json
// @Param request body CreateGroupRequest true "Create Group Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/create [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.groupService.CreateGroup(c.Request.Context(), currentUserID(c), req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "空间创建成功", group)
}

// JoinGroup godoc
// @Summary 加入空间
// @Description 使用邀请码加入已有空间
// @Tags Group
// @Accept json
// @Produce json
// @Param request body JoinGroupRequest true "Join Group Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/join [post]
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.groupService.JoinGroup(c.Request.Context(), currentUserID(c), req.InviteCode)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "加入空间成功", group)
}

// GetGroupInfo godoc
// @Summary 获取当前空间信息
// @Description 获取当前用户所在空间及成员信息
// @Tags Group
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/me [get]
func (h *GroupHandler) GetGroupInfo(c *gin.Context) {
	group, err := h.groupService.GetGroupInfo(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取空间成功", group)
}
