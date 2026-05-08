package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"vocalin-backend/internal/app"
	"vocalin-backend/internal/handlers"
	"vocalin-backend/internal/middleware"
	appvalidator "vocalin-backend/internal/validator"
)

func SetupRouter(application *app.App) (*gin.Engine, error) {
	if err := appvalidator.Register(); err != nil {
		return nil, fmt.Errorf("register validator: %w", err)
	}

	gin.SetMode(application.Config.Server.Mode)
	r := gin.New()
	r.Use(middleware.RequestLogger(application.Logger), gin.Recovery())

	// CORS
	r.Use(middleware.CORSMiddleware())

	authHandler := handlers.NewAuthHandler(application.Services.Auth)
	groupHandler := handlers.NewGroupHandler(application.Services.Group)
	homeHandler := handlers.NewHomeHandler(application.Services.Home)
	recordHandler := handlers.NewRecordHandler(application.Services.Record)
	profileHandler := handlers.NewProfileHandler(application.Services.Profile)

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": "SUCCESS", "message": "ok"})
	})

	// Auth
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	// Protected Routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(application.TokenManager))
	{
		api.POST("/auth/logout", authHandler.Logout)
		// Group
		api.GET("/groups", groupHandler.ListGroups)
		api.POST("/groups/create", groupHandler.CreateGroup)
		api.POST("/groups/join", groupHandler.JoinGroup)
		api.GET("/groups/current", groupHandler.GetGroupInfo)
		api.PUT("/groups/current", groupHandler.SwitchCurrentGroup)
		api.DELETE("/groups/:groupId/members/me", groupHandler.LeaveGroup)
		api.DELETE("/groups/:groupId/members/:userId", groupHandler.RemoveGroupMember)
		api.PUT("/groups/:groupId/owner", groupHandler.TransferGroupOwnership)
		api.POST("/groups/:groupId/join-requests/:requestId/review", groupHandler.ReviewJoinRequest)
		api.POST("/groups/:groupId/owner/review", groupHandler.ReviewOwnershipTransfer)
		api.DELETE("/groups/:groupId", groupHandler.DisbandGroup)

		// Home
		api.PUT("/home/status", homeHandler.UpdateStatus)
		api.PUT("/home/pinned", homeHandler.UpdatePinnedMessage)
		api.GET("/home/dashboard", homeHandler.GetHomeDashboard)
		api.GET("/home/messages", homeHandler.ListMessages)

		// Records
		api.POST("/records/photos", recordHandler.CreatePhoto)
		api.PUT("/records/photos/:id", recordHandler.UpdatePhoto)
		api.GET("/records/photos", recordHandler.GetPhotos)
		api.POST("/records/notes", recordHandler.CreateNote)
		api.PUT("/records/notes/:id", recordHandler.UpdateNote)
		api.GET("/records/notes", recordHandler.GetNotes)
		api.POST("/records/wishlist", recordHandler.CreateWishlist)
		api.PUT("/records/wishlist/:id", recordHandler.UpdateWishlist)
		api.GET("/records/wishlist", recordHandler.GetWishlist)
		api.PUT("/records/wishlist/:id/complete", recordHandler.CompleteWishlist)
		api.PUT("/records/wishlist/:id/incomplete", recordHandler.IncompleteWishlist)

		// Profile
		api.PUT("/profile/update", profileHandler.UpdateProfile)
		api.POST("/profile/anniversaries", profileHandler.CreateAnniversary)
		api.GET("/profile/anniversaries", profileHandler.GetAnniversaries)
		api.POST("/profile/leave", profileHandler.LeaveGroup)
		api.POST("/profile/export", profileHandler.ExportData)
	}

	return r, nil
}
