package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"
)

type ProfileHandler struct {
	profileService *service.ProfileService
}

type AnniversaryResponse = models.Anniversary

type CreateAnniversaryRequest struct {
	Title string    `json:"title" binding:"required,min=2,max=100"`
	Date  time.Time `json:"date" binding:"required"`
}

func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

// CreateAnniversary godoc
// @Summary 新增纪念日
// @Tags Profile
// @Accept json
// @Produce json
// @Param request body CreateAnniversaryRequest true "Create Anniversary Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=AnniversaryResponse}
// @Router /profile/anniversaries [post]
func (h *ProfileHandler) CreateAnniversary(c *gin.Context) {
	var req CreateAnniversaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	anniversary, err := h.profileService.CreateAnniversary(c.Request.Context(), currentUserID(c), req.Title, req.Date)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建纪念日成功", anniversary)
}

// GetAnniversaries godoc
// @Summary 获取纪念日列表
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Success 200 {object} response.APIResponse{data=[]AnniversaryResponse,meta=response.PaginationMeta}
// @Router /profile/anniversaries [get]
func (h *ProfileHandler) GetAnniversaries(c *gin.Context) {
	page, pageSize := parsePagination(c)
	anniversaries, err := h.profileService.ListAnniversaries(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取纪念日列表成功", anniversaries.Items, response.NewPaginationMeta(anniversaries.Page, anniversaries.PageSize, anniversaries.Total))
}

// LeaveGroup godoc
// @Summary 退出当前空间
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /profile/leave [post]
func (h *ProfileHandler) LeaveGroup(c *gin.Context) {
	if err := h.profileService.LeaveGroup(c.Request.Context(), currentUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "已成功退出空间", nil)
}

// ExportData godoc
// @Summary 导出个人数据
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /profile/export [post]
func (h *ProfileHandler) ExportData(c *gin.Context) {
	message, err := h.profileService.ExportData(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, message, nil)
}
