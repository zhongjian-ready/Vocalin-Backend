package handlers

import (
	"net/http"
	"strconv"
	"time"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type CreatePhotoRequest struct {
	URL         string `json:"url" binding:"required"`
	Description string `json:"description"`
}

type CreateNoteRequest struct {
	Content string     `json:"content" binding:"required"`
	Color   string     `json:"color"`
	Type    string     `json:"type"` // "normal", "burn", "timed"
	ShowAt  *time.Time `json:"show_at"`
}

type CreateWishlistRequest struct {
	Content string `json:"content" binding:"required"`
}

// CreatePhoto godoc
// @Summary Upload a photo
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreatePhotoRequest true "Create Photo Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Photo
// @Router /records/photos [post]
func CreatePhoto(c *gin.Context) {
	var req CreatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	photo := models.Photo{
		GroupID:     groupID,
		UploaderID:  user.ID,
		URL:         req.URL,
		Description: req.Description,
	}

	if err := database.Queries.CreatePhoto(&photo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create photo"})
		return
	}

	c.JSON(http.StatusOK, photo)
}

// GetPhotos godoc
// @Summary List photos
// @Tags Records
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.Photo
// @Router /records/photos [get]
func GetPhotos(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	photos, err := database.Queries.ListPhotosByGroup(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load photos"})
		return
	}

	c.JSON(http.StatusOK, photos)
}

// CreateNote godoc
// @Summary Create a sticky note
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateNoteRequest true "Create Note Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Note
// @Router /records/notes [post]
func CreateNote(c *gin.Context) {
	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	note := models.Note{
		GroupID:  groupID,
		AuthorID: user.ID,
		Content:  req.Content,
		Color:    req.Color,
		Type:     req.Type,
		ShowAt:   req.ShowAt,
	}

	if err := database.Queries.CreateNote(&note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create note"})
		return
	}

	c.JSON(http.StatusOK, note)
}

// GetNotes godoc
// @Summary List sticky notes
// @Tags Records
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.Note
// @Router /records/notes [get]
func GetNotes(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	// Filter out timed notes that shouldn't be shown yet
	notes, err := database.Queries.ListVisibleNotesByGroup(groupID, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load notes"})
		return
	}

	c.JSON(http.StatusOK, notes)
}

// CreateWishlist godoc
// @Summary Add wishlist item
// @Tags Records
// @Accept json
// @Produce json
// @Param request body CreateWishlistRequest true "Create Wishlist Request"
// @Security ApiKeyAuth
// @Success 200 {object} models.Wishlist
// @Router /records/wishlist [post]
func CreateWishlist(c *gin.Context) {
	var req CreateWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	item := models.Wishlist{
		GroupID: groupID,
		Content: req.Content,
	}

	if err := database.Queries.CreateWishlistItem(&item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wishlist item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// GetWishlist godoc
// @Summary List wishlist items
// @Tags Records
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.Wishlist
// @Router /records/wishlist [get]
func GetWishlist(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	items, err := database.Queries.ListWishlistByGroup(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load wishlist"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// CompleteWishlist godoc
// @Summary Mark wishlist item as completed
// @Tags Records
// @Produce json
// @Param id path int true "Wishlist Item ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.Wishlist
// @Router /records/wishlist/{id}/complete [put]
func CompleteWishlist(c *gin.Context) {
	_, groupID, ok := mustCurrentGroupUser(c)
	if !ok {
		return
	}

	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item id"})
		return
	}

	item, err := database.Queries.GetWishlistItemByID(uint(itemID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if item.GroupID != groupID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	now := time.Now()
	item.IsCompleted = true
	item.CompletedAt = &now
	if err := database.Queries.SaveWishlistItem(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete wishlist item"})
		return
	}

	c.JSON(http.StatusOK, *item)
}
