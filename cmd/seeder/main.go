package main

import (
	"log"
	"time"
	"vocalin-backend/internal/config"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
)

func main() {
	// Load Config
	cfg := config.LoadConfig()

	// Connect Database
	database.ConnectDB(cfg)

	log.Println("Starting data seeding...")

	// 1. Create Users
	user1 := models.User{
		WeChatID:        "wx_user_001",
		Nickname:        "Romeo",
		AvatarURL:       "https://api.dicebear.com/7.x/avataaars/svg?seed=Romeo",
		CurrentStatus:   "Thinking of you",
		StatusUpdatedAt: time.Now(),
	}
	user2 := models.User{
		WeChatID:        "wx_user_002",
		Nickname:        "Juliet",
		AvatarURL:       "https://api.dicebear.com/7.x/avataaars/svg?seed=Juliet",
		CurrentStatus:   "Happy",
		StatusUpdatedAt: time.Now(),
	}

	// Check if users exist
	var count int64
	database.DB.Model(&models.User{}).Where("wechat_id = ?", user1.WeChatID).Count(&count)
	if count == 0 {
		database.DB.Create(&user1)
		log.Printf("Created User: %s (ID: %d)", user1.Nickname, user1.ID)
	} else {
		database.DB.Where("wechat_id = ?", user1.WeChatID).First(&user1)
		log.Printf("User exists: %s (ID: %d)", user1.Nickname, user1.ID)
	}

	database.DB.Model(&models.User{}).Where("wechat_id = ?", user2.WeChatID).Count(&count)
	if count == 0 {
		database.DB.Create(&user2)
		log.Printf("Created User: %s (ID: %d)", user2.Nickname, user2.ID)
	} else {
		database.DB.Where("wechat_id = ?", user2.WeChatID).First(&user2)
		log.Printf("User exists: %s (ID: %d)", user2.Nickname, user2.ID)
	}

	// 2. Create Group
	group := models.Group{
		Name:                  "Love Nest",
		InviteCode:            "LOVE01",
		CreatorID:             user1.ID,
		TimerTitle:            "We've been together for",
		TimerStartDate:        time.Now().AddDate(0, -6, 0), // 6 months ago
		PinnedMessage:         "Dinner at 7 PM tonight! ❤️",
		PinnedMessageAuthorID: user1.ID,
	}

	database.DB.Model(&models.Group{}).Where("invite_code = ?", group.InviteCode).Count(&count)
	if count == 0 {
		database.DB.Create(&group)
		log.Printf("Created Group: %s (ID: %d)", group.Name, group.ID)
	} else {
		database.DB.Where("invite_code = ?", group.InviteCode).First(&group)
		log.Printf("Group exists: %s (ID: %d)", group.Name, group.ID)
	}

	// 3. Assign Users to Group
	user1.GroupID = &group.ID
	user2.GroupID = &group.ID
	database.DB.Save(&user1)
	database.DB.Save(&user2)
	log.Println("Assigned users to group")

	// 4. Create Records (Photos)
	photos := []models.Photo{
		{
			GroupID:     group.ID,
			UploaderID:  user1.ID,
			URL:         "https://images.unsplash.com/photo-1516589178581-a7870abd3645?q=80&w=600&auto=format&fit=crop",
			Description: "Our first trip together",
		},
		{
			GroupID:     group.ID,
			UploaderID:  user2.ID,
			URL:         "https://images.unsplash.com/photo-1529333166437-7750a6dd5a70?q=80&w=600&auto=format&fit=crop",
			Description: "Weekend vibes",
		},
	}

	for _, p := range photos {
		database.DB.Create(&p)
	}
	log.Printf("Created %d photos", len(photos))

	// 5. Create Notes
	notes := []models.Note{
		{
			GroupID:  group.ID,
			AuthorID: user1.ID,
			Content:  "Don't forget to buy milk",
			Color:    "#FFD700", // Gold
			Type:     "normal",
		},
		{
			GroupID:  group.ID,
			AuthorID: user2.ID,
			Content:  "I love you!",
			Color:    "#FF69B4", // HotPink
			Type:     "normal",
		},
	}
	for _, n := range notes {
		database.DB.Create(&n)
	}
	log.Printf("Created %d notes", len(notes))

	// 6. Create Wishlist
	wishes := []models.Wishlist{
		{
			GroupID:     group.ID,
			Content:     "Visit Japan",
			IsCompleted: false,
		},
		{
			GroupID:     group.ID,
			Content:     "Watch the new Marvel movie",
			IsCompleted: true,
		},
	}
	for _, w := range wishes {
		database.DB.Create(&w)
	}
	log.Printf("Created %d wishlist items", len(wishes))

	// 7. Create Anniversaries
	anniversaries := []models.Anniversary{
		{
			UserID:  user1.ID,
			GroupID: group.ID,
			Title:   "First Date",
			Date:    time.Now().AddDate(-1, 0, 0), // 1 year ago
		},
		{
			UserID:  user2.ID,
			GroupID: group.ID,
			Title:   "Romeo's Birthday",
			Date:    time.Now().AddDate(0, 1, 5), // In 1 month and 5 days
		},
	}
	for _, a := range anniversaries {
		database.DB.Create(&a)
	}
	log.Printf("Created %d anniversaries", len(anniversaries))

	log.Println("Seeding completed successfully!")
}
