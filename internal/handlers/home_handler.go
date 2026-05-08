package handlers

import (
	"github.com/gin-gonic/gin"

	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"
)

type HomeHandler struct {
	homeService *service.HomeService
}

type HomeGroupResponse = models.Group
type HomeUserResponse = models.User
type HomeMessageListResponse = []service.MessageListItem

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,max=120"`
}

type UpdatePinnedMessageRequest struct {
	Content string `json:"content" binding:"required,max=500"`
}

func NewHomeHandler(homeService *service.HomeService) *HomeHandler {
	return &HomeHandler{homeService: homeService}
}

// UpdateStatus godoc
// @Summary 更新实时状态
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdateStatusRequest true "Update Status Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=HomeUserResponse}
// @Router /home/status [put]
func (h *HomeHandler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	user, err := h.homeService.UpdateStatus(c.Request.Context(), currentUserID(c), req.Status)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新状态成功", user)
}

// UpdatePinnedMessage godoc
// @Summary 更新置顶留言
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdatePinnedMessageRequest true "Update Pinned Message Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=HomeGroupResponse}
// @Router /home/pinned [put]
func (h *HomeHandler) UpdatePinnedMessage(c *gin.Context) {
	var req UpdatePinnedMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.homeService.UpdatePinnedMessage(c.Request.Context(), currentUserID(c), req.Content)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新置顶留言成功", group)
}

// GetHomeDashboard godoc
// @Summary 获取首页概览
// @Description 获取当前首页空间信息、最近动态，以及当前用户待处理消息数量
// @Tags Home
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=service.DashboardResult}
// @Router /home/dashboard [get]
func (h *HomeHandler) GetHomeDashboard(c *gin.Context) {
	result, err := h.homeService.GetDashboard(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取首页概览成功", result)
}

// ListMessages godoc
// @Summary 获取首页消息列表
// @Description 获取当前用户待处理的消息列表，包含加入申请和管理员移交申请
// @Tags Home
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=HomeMessageListResponse}
// @Router /home/messages [get]
func (h *HomeHandler) ListMessages(c *gin.Context) {
	result, err := h.homeService.ListMessages(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取首页消息成功", result)
}
