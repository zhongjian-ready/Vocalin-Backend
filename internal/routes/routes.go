package routes

import (
	"vocalin-backend/internal/handlers"
	"vocalin-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS
	r.Use(middleware.CORSMiddleware())

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Auth
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", handlers.Login)
	}

	// Protected Routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Group
		api.POST("/groups", handlers.CreateGroup)
		api.POST("/groups/join", handlers.JoinGroup)
		api.GET("/groups/me", handlers.GetGroupInfo)

		// Home
		api.PUT("/home/timer", handlers.UpdateTimer)
		api.PUT("/home/status", handlers.UpdateStatus)
		api.PUT("/home/pinned", handlers.UpdatePinnedMessage)
		api.GET("/home/dashboard", handlers.GetHomeDashboard)

		// Records
		api.POST("/records/photos", handlers.CreatePhoto)
		api.GET("/records/photos", handlers.GetPhotos)
		api.POST("/records/notes", handlers.CreateNote)
		api.GET("/records/notes", handlers.GetNotes)
		api.POST("/records/wishlist", handlers.CreateWishlist)
		api.GET("/records/wishlist", handlers.GetWishlist)
		api.PUT("/records/wishlist/:id/complete", handlers.CompleteWishlist)

		// Profile
		api.POST("/profile/anniversaries", handlers.CreateAnniversary)
		api.GET("/profile/anniversaries", handlers.GetAnniversaries)
		api.POST("/profile/leave", handlers.LeaveGroup)
		api.POST("/profile/export", handlers.ExportData)
	}

	return r
}
