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

	if err := database.Queries.EnsureUserByWeChatID(&user1); err != nil {
		log.Fatalf("Failed to seed user %s: %v", user1.WeChatID, err)
	}
	log.Printf("Ensured User: %s (ID: %d)", user1.Nickname, user1.ID)

	if err := database.Queries.EnsureUserByWeChatID(&user2); err != nil {
		log.Fatalf("Failed to seed user %s: %v", user2.WeChatID, err)
	}
	log.Printf("Ensured User: %s (ID: %d)", user2.Nickname, user2.ID)

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

	if err := database.Queries.EnsureGroupByInviteCode(&group); err != nil {
		log.Fatalf("Failed to seed group %s: %v", group.InviteCode, err)
	}
	log.Printf("Ensured Group: %s (ID: %d)", group.Name, group.ID)

	// 3. Assign Users to Group
	user1.GroupID = &group.ID
	user2.GroupID = &group.ID
	if err := database.Queries.SaveUser(&user1); err != nil {
		log.Fatalf("Failed to assign user %d to group: %v", user1.ID, err)
	}
	if err := database.Queries.SaveUser(&user2); err != nil {
		log.Fatalf("Failed to assign user %d to group: %v", user2.ID, err)
	}
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
		if err := database.Queries.CreatePhoto(&p); err != nil {
			log.Fatalf("Failed to seed photo: %v", err)
		}
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
		if err := database.Queries.CreateNote(&n); err != nil {
			log.Fatalf("Failed to seed note: %v", err)
		}
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
		if err := database.Queries.CreateWishlistItem(&w); err != nil {
			log.Fatalf("Failed to seed wishlist item: %v", err)
		}
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
		if err := database.Queries.CreateAnniversary(&a); err != nil {
			log.Fatalf("Failed to seed anniversary: %v", err)
		}
	}
	log.Printf("Created %d anniversaries", len(anniversaries))

	log.Println("Seeding completed successfully!")
}
