package handlers

import (
	"net/http"
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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	photo := models.Photo{
		GroupID:     *user.GroupID,
		UploaderID:  userID,
		URL:         req.URL,
		Description: req.Description,
	}

	if err := database.DB.Create(&photo).Error; err != nil {
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
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var photos []models.Photo
	database.DB.Where("group_id = ?", *user.GroupID).Preload("Comments").Preload("Likes").Order("created_at desc").Find(&photos)

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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	note := models.Note{
		GroupID:  *user.GroupID,
		AuthorID: userID,
		Content:  req.Content,
		Color:    req.Color,
		Type:     req.Type,
		ShowAt:   req.ShowAt,
	}

	if err := database.DB.Create(&note).Error; err != nil {
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
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var notes []models.Note
	// Filter out timed notes that shouldn't be shown yet
	now := time.Now()
	database.DB.Where("group_id = ? AND (show_at IS NULL OR show_at <= ?)", *user.GroupID, now).Order("created_at desc").Find(&notes)

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

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	item := models.Wishlist{
		GroupID: *user.GroupID,
		Content: req.Content,
	}

	if err := database.DB.Create(&item).Error; err != nil {
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
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if user.GroupID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not in a group"})
		return
	}

	var items []models.Wishlist
	database.DB.Where("group_id = ?", *user.GroupID).Order("created_at desc").Find(&items)

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
	id := c.Param("id")
	var item models.Wishlist
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	now := time.Now()
	item.IsCompleted = true
	item.CompletedAt = &now
	database.DB.Save(&item)

	c.JSON(http.StatusOK, item)
}
