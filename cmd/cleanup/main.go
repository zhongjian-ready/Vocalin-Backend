package main

import (
	"log"
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
)

func main() {
	// Load Config
	cfg := config.LoadConfig()

	// Connect Database
	database.ConnectDB(cfg)

	log.Println("Starting database cleanup...")

	// Check and drop 'we_chat_id' column from 'users' table if it exists
	if database.DB.Migrator().HasColumn(&models.User{}, "we_chat_id") {
		log.Println("Found deprecated column 'we_chat_id' in 'users' table. Dropping it...")
		if err := database.DB.Migrator().DropColumn(&models.User{}, "we_chat_id"); err != nil {
			log.Fatalf("Failed to drop column 'we_chat_id': %v", err)
		}
		log.Println("Successfully dropped column 'we_chat_id'.")
	} else {
		log.Println("Column 'we_chat_id' does not exist in 'users' table.")
	}

	log.Println("Database cleanup completed.")
}
