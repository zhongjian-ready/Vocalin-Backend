package main

import (
	"log"
	_ "vocalin-backend/docs" // Import generated docs
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/routes"
)

// @title Vocalin API
// @version 1.0
// @description API for Vocalin (窝聚) App
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-User-ID
func main() {
	// Load Config
	cfg := config.LoadConfig()

	// Connect Database
	database.ConnectDB(cfg)

	// Auto Migrate
	err := database.DB.AutoMigrate(
		&models.Group{},
		&models.User{},
		&models.Photo{},
		&models.Comment{},
		&models.Like{},
		&models.Note{},
		&models.Wishlist{},
		&models.Anniversary{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	// Setup Router
	r := routes.SetupRouter()

	// Run Server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
