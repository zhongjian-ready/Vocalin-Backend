package main

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"vocalin-backend/internal/app"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = application.Close() }()

	ctx := context.Background()
	store := application.Store

	if err := database.AutoMigrate(application.DB); err != nil {
		application.Logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	application.Logger.Info("开始写入示例数据")

	// 1. Create Users
	user1 := models.User{
		WeChatID:        "wechat-romeo",
		Nickname:        "Romeo",
		AvatarURL:       "https://api.dicebear.com/7.x/avataaars/svg?seed=Romeo",
		CurrentStatus:   "Thinking of you",
		StatusUpdatedAt: time.Now(),
	}
	user2 := models.User{
		WeChatID:        "wechat-juliet",
		Nickname:        "Juliet",
		AvatarURL:       "https://api.dicebear.com/7.x/avataaars/svg?seed=Juliet",
		CurrentStatus:   "Happy",
		StatusUpdatedAt: time.Now(),
	}

	if err := ensureUserByWeChatID(ctx, store, &user1); err != nil {
		application.Logger.Fatal("写入用户失败", zap.Error(err))
	}
	application.Logger.Info("用户已准备就绪", zap.String("nickname", user1.Nickname))

	if err := ensureUserByWeChatID(ctx, store, &user2); err != nil {
		application.Logger.Fatal("写入用户失败", zap.Error(err))
	}
	application.Logger.Info("用户已准备就绪", zap.String("nickname", user2.Nickname))

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

	if err := store.EnsureGroupByInviteCode(ctx, &group); err != nil {
		application.Logger.Fatal("写入空间失败", zap.Error(err))
	}
	application.Logger.Info("空间已准备就绪", zap.String("group", group.Name))

	// 3. Assign Users to Group
	user1.GroupID = &group.ID
	user2.GroupID = &group.ID
	if err := store.SaveUser(ctx, &user1); err != nil {
		application.Logger.Fatal("更新用户空间失败", zap.Error(err))
	}
	if err := store.SaveUser(ctx, &user2); err != nil {
		application.Logger.Fatal("更新用户空间失败", zap.Error(err))
	}
	application.Logger.Info("用户已加入空间")

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
		if err := store.CreatePhoto(ctx, &p); err != nil {
			application.Logger.Fatal("写入照片失败", zap.Error(err))
		}
	}
	application.Logger.Info("照片数据写入完成")

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
		if err := store.CreateNote(ctx, &n); err != nil {
			application.Logger.Fatal("写入便签失败", zap.Error(err))
		}
	}
	application.Logger.Info("便签数据写入完成")

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
		if err := store.CreateWishlistItem(ctx, &w); err != nil {
			application.Logger.Fatal("写入愿望清单失败", zap.Error(err))
		}
	}
	application.Logger.Info("愿望清单数据写入完成")

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
		if err := store.CreateAnniversary(ctx, &a); err != nil {
			application.Logger.Fatal("写入纪念日失败", zap.Error(err))
		}
	}
	application.Logger.Info("纪念日数据写入完成")

	application.Logger.Info("示例数据写入完成")
}

func ensureUserByWeChatID(ctx context.Context, store interface {
	GetUserByWeChatID(context.Context, string) (*models.User, error)
	CreateUser(context.Context, *models.User) error
	SaveUser(context.Context, *models.User) error
}, user *models.User) error {
	existing, err := store.GetUserByWeChatID(ctx, user.WeChatID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return store.CreateUser(ctx, user)
	}

	existing.Nickname = user.Nickname
	existing.AvatarURL = user.AvatarURL
	existing.CurrentStatus = user.CurrentStatus
	existing.StatusUpdatedAt = user.StatusUpdatedAt
	user.Model = existing.Model
	user.GroupID = existing.GroupID
	return store.SaveUser(ctx, existing)
}
