package handlers

import (
	"strconv"
	"time"
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type RecordHandler struct {
	recordService *service.RecordService
}

type PhotoResponse = models.Photo
type NoteResponse = models.Note
type WishlistResponse = models.Wishlist

type CreatePhotoRequest struct {
	URL         string `json:"url" binding:"required,url,max=1024"`
	Description string `json:"description" binding:"max=500"`
	Visibility  string `json:"visibility" binding:"omitempty,oneof=public private"`
}

type CreateNoteRequest struct {
	Content    string     `json:"content" binding:"required,max=1000"`
	Color      string     `json:"color" binding:"max=20"`
	Type       string     `json:"type" binding:"required,note_type"`
	ShowAt     *time.Time `json:"show_at"`
	Visibility string     `json:"visibility" binding:"omitempty,oneof=public private"`
}

type CreateWishlistRequest struct {
	Content    string `json:"content" binding:"required,max=255"`
	Priority   string `json:"priority" binding:"omitempty,oneof=low medium high"`
	Visibility string `json:"visibility" binding:"omitempty,oneof=public private"`
}

type UpdatePhotoRequest = CreatePhotoRequest

type UpdateNoteRequest = CreateNoteRequest

type UpdateWishlistRequest = CreateWishlistRequest

func NewRecordHandler(recordService *service.RecordService) *RecordHandler {
	return &RecordHandler{recordService: recordService}
}

// CreatePhoto godoc
// @Summary 上传照片记录
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreatePhotoRequest true "Create Photo Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=PhotoResponse}
// @Router /records/photos [post]
func (h *RecordHandler) CreatePhoto(c *gin.Context) {
	var req CreatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	photo, err := h.recordService.CreatePhoto(c.Request.Context(), currentUserID(c), req.URL, req.Description, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建照片成功", photo)
}

// UpdatePhoto godoc
// @Summary 编辑照片记录
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Photo ID"
// @Param request body UpdatePhotoRequest true "Update Photo Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=PhotoResponse}
// @Router /records/photos/{id} [put]
func (h *RecordHandler) UpdatePhoto(c *gin.Context) {
	photoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的照片 ID")
		return
	}

	var req UpdatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	photo, err := h.recordService.UpdatePhoto(c.Request.Context(), currentUserID(c), uint(photoID), req.URL, req.Description, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新照片成功", photo)
}

// GetPhotos godoc
// @Summary 获取照片列表
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Success 200 {object} response.APIResponse{data=[]PhotoResponse,meta=response.PaginationMeta}
// @Router /records/photos [get]
func (h *RecordHandler) GetPhotos(c *gin.Context) {
	page, pageSize := parsePagination(c)
	photos, err := h.recordService.ListPhotos(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取照片列表成功", photos.Items, response.NewPaginationMeta(photos.Page, photos.PageSize, photos.Total))
}

// CreateNote godoc
// @Summary 创建便签
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateNoteRequest true "Create Note Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteResponse}
// @Router /records/notes [post]
func (h *RecordHandler) CreateNote(c *gin.Context) {
	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	note, err := h.recordService.CreateNote(c.Request.Context(), currentUserID(c), req.Content, req.Color, req.Type, req.ShowAt, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建便签成功", note)
}

// UpdateNote godoc
// @Summary 编辑便签
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Note ID"
// @Param request body UpdateNoteRequest true "Update Note Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteResponse}
// @Router /records/notes/{id} [put]
func (h *RecordHandler) UpdateNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签 ID")
		return
	}

	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	note, err := h.recordService.UpdateNote(c.Request.Context(), currentUserID(c), uint(noteID), req.Content, req.Color, req.Type, req.ShowAt, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新便签成功", note)
}

// GetNotes godoc
// @Summary 获取便签列表
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Success 200 {object} response.APIResponse{data=[]NoteResponse,meta=response.PaginationMeta}
// @Router /records/notes [get]
func (h *RecordHandler) GetNotes(c *gin.Context) {
	page, pageSize := parsePagination(c)
	notes, err := h.recordService.ListNotes(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取便签列表成功", notes.Items, response.NewPaginationMeta(notes.Page, notes.PageSize, notes.Total))
}

// CreateWishlist godoc
// @Summary 新增愿望清单项
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateWishlistRequest true "Create Wishlist Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=WishlistResponse}
// @Router /records/wishlist [post]
func (h *RecordHandler) CreateWishlist(c *gin.Context) {
	var req CreateWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	item, err := h.recordService.CreateWishlist(c.Request.Context(), currentUserID(c), req.Content, req.Priority, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建愿望清单成功", item)
}

// UpdateWishlist godoc
// @Summary 编辑愿望清单项
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Wishlist Item ID"
// @Param request body UpdateWishlistRequest true "Update Wishlist Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=WishlistResponse}
// @Router /records/wishlist/{id} [put]
func (h *RecordHandler) UpdateWishlist(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的愿望清单 ID")
		return
	}

	var req UpdateWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	item, err := h.recordService.UpdateWishlist(c.Request.Context(), currentUserID(c), uint(itemID), req.Content, req.Priority, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新愿望清单成功", item)
}

// GetWishlist godoc
// @Summary 获取愿望清单
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Success 200 {object} response.APIResponse{data=[]WishlistResponse,meta=response.PaginationMeta}
// @Router /records/wishlist [get]
func (h *RecordHandler) GetWishlist(c *gin.Context) {
	page, pageSize := parsePagination(c)
	items, err := h.recordService.ListWishlist(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取愿望清单成功", items.Items, response.NewPaginationMeta(items.Page, items.PageSize, items.Total))
}

// CompleteWishlist godoc
// @Summary 完成愿望清单项
// @Tags Records
// @Produce json
// @Param id path int true "Wishlist Item ID"
// @Security BearerAuth
// @Success 200 {object} WishlistResponse
// @Router /records/wishlist/{id}/complete [put]
func (h *RecordHandler) CompleteWishlist(c *gin.Context) {
	h.updateWishlistCompletion(c, true)
}

// IncompleteWishlist godoc
// @Summary 取消完成愿望清单项
// @Tags Records
// @Produce json
// @Param id path int true "Wishlist Item ID"
// @Security BearerAuth
// @Success 200 {object} WishlistResponse
// @Router /records/wishlist/{id}/incomplete [put]
func (h *RecordHandler) IncompleteWishlist(c *gin.Context) {
	h.updateWishlistCompletion(c, false)
}

func (h *RecordHandler) updateWishlistCompletion(c *gin.Context, completed bool) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的愿望清单 ID")
		return
	}

	var item *models.Wishlist
	if completed {
		item, err = h.recordService.CompleteWishlist(c.Request.Context(), currentUserID(c), uint(itemID))
	} else {
		item, err = h.recordService.IncompleteWishlist(c.Request.Context(), currentUserID(c), uint(itemID))
	}
	if err != nil {
		writeServiceError(c, err)
		return
	}

	message := "完成愿望清单成功"
	if !completed {
		message = "取消完成愿望清单成功"
	}

	response.Success(c, message, item)
}
