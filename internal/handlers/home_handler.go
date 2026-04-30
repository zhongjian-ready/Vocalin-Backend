package handlers

import (
	"time"

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

type UpdateTimerRequest struct {
	Title     string    `json:"title" binding:"required,min=2,max=100"`
	StartDate time.Time `json:"start_date" binding:"required"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,max=120"`
}

type UpdatePinnedMessageRequest struct {
	Content string `json:"content" binding:"required,max=500"`
}

func NewHomeHandler(homeService *service.HomeService) *HomeHandler {
	return &HomeHandler{homeService: homeService}
}

// UpdateTimer godoc
// @Summary 更新陪伴计时器
// @Tags Home
// @Accept json
// @Produce json
// @Param request body UpdateTimerRequest true "Update Timer Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=HomeGroupResponse}
// @Router /home/timer [put]
func (h *HomeHandler) UpdateTimer(c *gin.Context) {
	var req UpdateTimerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.homeService.UpdateTimer(c.Request.Context(), currentUserID(c), req.Title, req.StartDate)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新计时器成功", group)
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
