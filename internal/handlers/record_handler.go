package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecordHandler struct {
	recordService *service.RecordService
}

type AlbumResponse struct {
	gorm.Model
	GroupID     uint                 `json:"group_id"`
	CreatorID   uint                 `json:"creator_id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Visibility  string               `json:"visibility"`
	Photos      []AlbumPhotoResponse `json:"photos"`
	Comments    []models.Comment     `json:"comments"`
	Likes       int                  `json:"likes"`
}

type AlbumPhotoResponse struct {
	gorm.Model
	AlbumID    uint   `json:"album_id"`
	GroupID    uint   `json:"group_id"`
	UploaderID uint   `json:"uploader_id"`
	URL        string `json:"url"`
}

type NoteResponse struct {
	gorm.Model
	GroupID    uint        `json:"group_id"`
	AuthorID   uint        `json:"author_id"`
	FolderID   *uint       `json:"folder_id,omitempty"`
	FolderName string      `json:"folder_name,omitempty"`
	FolderType string      `json:"folder_type"`
	Content    string      `json:"content"`
	Color      string      `json:"color"`
	Type       string      `json:"type"`
	ShowAt     *time.Time  `json:"show_at"`
	Visibility string      `json:"visibility"`
	IsBurned   bool        `json:"is_burned"`
	Author     models.User `json:"author"`
}

type NoteFolderResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Editable  bool   `json:"editable"`
	Deletable bool   `json:"deletable"`
}

type WishlistResponse = models.Wishlist

type AlbumPhotoRequest struct {
	URL string `json:"url" binding:"required,url,max=1024"`
}

type CreateAlbumRequest struct {
	Title       string              `json:"title" binding:"required,max=255"`
	Description string              `json:"description" binding:"max=500"`
	Visibility  string              `json:"visibility" binding:"omitempty,oneof=public private"`
	Photos      []AlbumPhotoRequest `json:"photos" binding:"required,min=1,dive"`
}

type CreateNoteRequest struct {
	FolderID   *uint      `json:"folder_id"`
	Content    string     `json:"content" binding:"required"`
	Color      string     `json:"color" binding:"max=20"`
	Type       string     `json:"type" binding:"required,note_type"`
	ShowAt     *time.Time `json:"show_at"`
	Visibility string     `json:"visibility" binding:"omitempty,oneof=public private"`
}

type CreateNoteFolderRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

type MoveNoteRequest struct {
	FolderID *uint `json:"folder_id"`
}

type UpdateNoteVisibilityRequest struct {
	Visibility string `json:"visibility" binding:"required,oneof=public private"`
}

type CreateWishlistRequest struct {
	Content    string `json:"content" binding:"required,max=255"`
	Priority   string `json:"priority" binding:"omitempty,oneof=low medium high"`
	Visibility string `json:"visibility" binding:"omitempty,oneof=public private"`
}

type UpdateAlbumRequest = CreateAlbumRequest

type UpdateNoteRequest = CreateNoteRequest

type UpdateNoteFolderRequest = CreateNoteFolderRequest

type UpdateWishlistRequest = CreateWishlistRequest

func NewRecordHandler(recordService *service.RecordService) *RecordHandler {
	return &RecordHandler{recordService: recordService}
}

// CreateAlbum godoc
// @Summary 创建相册
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateAlbumRequest true "Create Album Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=AlbumResponse}
// @Router /records/albums [post]
func (h *RecordHandler) CreateAlbum(c *gin.Context) {
	var req CreateAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	album, err := h.recordService.CreateAlbum(c.Request.Context(), currentUserID(c), req.Title, req.Description, req.Visibility, toAlbumPhotoInputs(req.Photos))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建相册成功", toAlbumResponse(album))
}

// UpdateAlbum godoc
// @Summary 编辑相册
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Album ID"
// @Param request body UpdateAlbumRequest true "Update Album Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=AlbumResponse}
// @Router /records/albums/{id} [put]
func (h *RecordHandler) UpdateAlbum(c *gin.Context) {
	albumID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的相册 ID")
		return
	}

	var req UpdateAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	album, err := h.recordService.UpdateAlbum(c.Request.Context(), currentUserID(c), uint(albumID), req.Title, req.Description, req.Visibility, toAlbumPhotoInputs(req.Photos))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新相册成功", toAlbumResponse(album))
}

// GetAlbums godoc
// @Summary 获取相册列表
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Success 200 {object} response.APIResponse{data=[]AlbumResponse,meta=response.PaginationMeta}
// @Router /records/albums [get]
func (h *RecordHandler) GetAlbums(c *gin.Context) {
	page, pageSize := parsePagination(c)
	albums, err := h.recordService.ListAlbums(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取相册列表成功", toAlbumResponses(albums.Items), response.NewPaginationMeta(albums.Page, albums.PageSize, albums.Total))
}

// DeleteAlbum godoc
// @Summary 删除相册
// @Tags Records
// @Produce json
// @Param id path int true "Album ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /records/albums/{id} [delete]
func (h *RecordHandler) DeleteAlbum(c *gin.Context) {
	albumID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的相册 ID")
		return
	}

	if err := h.recordService.DeleteAlbum(c.Request.Context(), currentUserID(c), uint(albumID)); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "删除相册成功", nil)
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

	note, err := h.recordService.CreateNote(c.Request.Context(), currentUserID(c), req.FolderID, req.Content, req.Color, req.Type, req.ShowAt, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建便签成功", toNoteResponse(note, currentUserID(c)))
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

	note, err := h.recordService.UpdateNote(c.Request.Context(), currentUserID(c), uint(noteID), req.FolderID, req.Content, req.Color, req.Type, req.ShowAt, req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新便签成功", toNoteResponse(note, currentUserID(c)))
}

// GetNotes godoc
// @Summary 获取便签列表
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，从 1 开始"
// @Param page_size query int false "每页条数，最大 100"
// @Param folder_type query string false "分类过滤：all、shared、custom"
// @Param folder_id query int false "自定义分类 ID，传入后精确过滤该分类"
// @Success 200 {object} response.APIResponse{data=[]NoteResponse,meta=response.PaginationMeta}
// @Router /records/notes [get]
func (h *RecordHandler) GetNotes(c *gin.Context) {
	page, pageSize := parsePagination(c)
	filter, err := parseNoteListFilter(c)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	notes, err := h.recordService.ListNotes(c.Request.Context(), currentUserID(c), service.NewPagination(page, pageSize), filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.JSON(c, 200, "SUCCESS", "获取便签列表成功", toNoteResponses(notes.Items, currentUserID(c)), response.NewPaginationMeta(notes.Page, notes.PageSize, notes.Total))
}

// DeleteNote godoc
// @Summary 删除便签
// @Tags Records
// @Produce json
// @Param id path int true "Note ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /records/notes/{id} [delete]
func (h *RecordHandler) DeleteNote(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签 ID")
		return
	}

	if err := h.recordService.DeleteNote(c.Request.Context(), currentUserID(c), uint(noteID)); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "删除便签成功", nil)
}

// MoveNoteToFolder godoc
// @Summary 移动便签到分类
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Note ID"
// @Param request body MoveNoteRequest true "Move Note Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteResponse}
// @Router /records/notes/{id}/folder [put]
func (h *RecordHandler) MoveNoteToFolder(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签 ID")
		return
	}

	var req MoveNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	note, err := h.recordService.MoveNoteToFolder(c.Request.Context(), currentUserID(c), uint(noteID), req.FolderID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "移动便签分类成功", toNoteResponse(note, currentUserID(c)))
}

// UpdateNoteVisibility godoc
// @Summary 更新便签可见性
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Note ID"
// @Param request body UpdateNoteVisibilityRequest true "Update Note Visibility Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteResponse}
// @Router /records/notes/{id}/visibility [put]
func (h *RecordHandler) UpdateNoteVisibility(c *gin.Context) {
	noteID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签 ID")
		return
	}

	var req UpdateNoteVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	note, err := h.recordService.UpdateNoteVisibility(c.Request.Context(), currentUserID(c), uint(noteID), req.Visibility)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新便签可见性成功", toNoteResponse(note, currentUserID(c)))
}

// CreateNoteFolder godoc
// @Summary 创建便签分类
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateNoteFolderRequest true "Create Note Folder Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteFolderResponse}
// @Router /records/note-folders [post]
func (h *RecordHandler) CreateNoteFolder(c *gin.Context) {
	var req CreateNoteFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	folder, err := h.recordService.CreateNoteFolder(c.Request.Context(), currentUserID(c), req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "创建便签分类成功", toNoteFolderResponse(folder))
}

// UpdateNoteFolder godoc
// @Summary 编辑便签分类
// @Tags Records
// @Accept json
// @Produce json
// @Param id path int true "Note Folder ID"
// @Param request body UpdateNoteFolderRequest true "Update Note Folder Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=NoteFolderResponse}
// @Router /records/note-folders/{id} [put]
func (h *RecordHandler) UpdateNoteFolder(c *gin.Context) {
	folderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签分类 ID")
		return
	}

	var req UpdateNoteFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	folder, err := h.recordService.UpdateNoteFolder(c.Request.Context(), currentUserID(c), uint(folderID), req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "更新便签分类成功", toNoteFolderResponse(folder))
}

// DeleteNoteFolder godoc
// @Summary 删除便签分类
// @Tags Records
// @Produce json
// @Param id path int true "Note Folder ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /records/note-folders/{id} [delete]
func (h *RecordHandler) DeleteNoteFolder(c *gin.Context) {
	folderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的便签分类 ID")
		return
	}

	if err := h.recordService.DeleteNoteFolder(c.Request.Context(), currentUserID(c), uint(folderID)); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "删除便签分类成功", nil)
}

// GetNoteFolders godoc
// @Summary 获取便签分类列表
// @Tags Records
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=[]NoteFolderResponse}
// @Router /records/note-folders [get]
func (h *RecordHandler) GetNoteFolders(c *gin.Context) {
	folders, err := h.recordService.ListNoteFolders(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取便签分类成功", toNoteFolderResponses(folders))
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

// DeleteWishlist godoc
// @Summary 删除愿望清单项
// @Tags Records
// @Produce json
// @Param id path int true "Wishlist Item ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /records/wishlist/{id} [delete]
func (h *RecordHandler) DeleteWishlist(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "VALIDATION_ERROR", "无效的愿望清单 ID")
		return
	}

	if err := h.recordService.DeleteWishlist(c.Request.Context(), currentUserID(c), uint(itemID)); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "删除愿望清单成功", nil)
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

func toAlbumPhotoInputs(photos []AlbumPhotoRequest) []service.AlbumPhotoInput {
	inputs := make([]service.AlbumPhotoInput, 0, len(photos))
	for _, photo := range photos {
		inputs = append(inputs, service.AlbumPhotoInput{
			URL: photo.URL,
		})
	}
	return inputs
}

func parseNoteListFilter(c *gin.Context) (service.NoteListFilter, error) {
	filter := service.NoteListFilter{FolderType: strings.TrimSpace(strings.ToLower(c.Query("folder_type")))}
	if filter.FolderType != "" && filter.FolderType != "all" && filter.FolderType != "shared" && filter.FolderType != "custom" {
		return service.NoteListFilter{}, fmt.Errorf("folder_type 仅支持 all、shared、custom")
	}

	folderIDParam := strings.TrimSpace(c.Query("folder_id"))
	if folderIDParam == "" {
		return filter, nil
	}

	parsed, err := strconv.ParseUint(folderIDParam, 10, 64)
	if err != nil || parsed == 0 {
		return service.NoteListFilter{}, fmt.Errorf("folder_id 必须为有效的正整数")
	}
	folderID := uint(parsed)
	filter.FolderID = &folderID
	return filter, nil
}

func toNoteResponses(notes []models.Note, viewerID uint) []NoteResponse {
	items := make([]NoteResponse, 0, len(notes))
	for _, note := range notes {
		items = append(items, toNoteResponse(&note, viewerID))
	}
	return items
}

func toNoteResponse(note *models.Note, viewerID uint) NoteResponse {
	item := NoteResponse{
		Model: gorm.Model{
			ID:        note.ID,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
			DeletedAt: note.DeletedAt,
		},
		GroupID:    note.GroupID,
		AuthorID:   note.AuthorID,
		Content:    note.Content,
		Color:      note.Color,
		Type:       note.Type,
		ShowAt:     note.ShowAt,
		Visibility: note.Visibility,
		IsBurned:   note.IsBurned,
		Author:     note.Author,
	}

	if note.AuthorID != viewerID {
		item.FolderType = "shared"
		return item
	}
	if note.FolderID != nil && note.Folder != nil {
		item.FolderID = note.FolderID
		item.FolderName = note.Folder.Name
		item.FolderType = "custom"
		return item
	}
	item.FolderType = "all"
	return item
}

func toNoteFolderResponses(folders []models.NoteFolder) []NoteFolderResponse {
	items := []NoteFolderResponse{
		{ID: 0, Name: "All", Type: "all", Editable: false, Deletable: false},
		{ID: 0, Name: "Shared", Type: "shared", Editable: false, Deletable: false},
	}
	for _, folder := range folders {
		items = append(items, toNoteFolderResponse(&folder))
	}
	return items
}

func toNoteFolderResponse(folder *models.NoteFolder) NoteFolderResponse {
	return NoteFolderResponse{ID: folder.ID, Name: folder.Name, Type: "custom", Editable: true, Deletable: true}
}

func toAlbumResponses(albums []models.Album) []AlbumResponse {
	items := make([]AlbumResponse, 0, len(albums))
	for _, album := range albums {
		items = append(items, toAlbumResponse(&album))
	}
	return items
}

func toAlbumResponse(album *models.Album) AlbumResponse {
	photos := make([]AlbumPhotoResponse, 0, len(album.Photos))
	for _, photo := range album.Photos {
		photos = append(photos, AlbumPhotoResponse{
			Model: gorm.Model{
				ID:        photo.ID,
				CreatedAt: photo.CreatedAt,
				UpdatedAt: photo.UpdatedAt,
				DeletedAt: photo.DeletedAt,
			},
			AlbumID:    photo.AlbumID,
			GroupID:    photo.GroupID,
			UploaderID: photo.UploaderID,
			URL:        photo.URL,
		})
	}

	comments := make([]models.Comment, len(album.Comments))
	copy(comments, album.Comments)

	return AlbumResponse{
		Model: gorm.Model{
			ID:        album.ID,
			CreatedAt: album.CreatedAt,
			UpdatedAt: album.UpdatedAt,
			DeletedAt: album.DeletedAt,
		},
		GroupID:     album.GroupID,
		CreatorID:   album.CreatorID,
		Title:       album.Title,
		Description: album.Description,
		Visibility:  album.Visibility,
		Photos:      photos,
		Comments:    comments,
		Likes:       len(album.Likes),
	}
}
