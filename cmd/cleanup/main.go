package main

import (
	"vocalin-backend/internal/app"
	"vocalin-backend/internal/database"
	"vocalin-backend/internal/models"

	"go.uber.org/zap"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}
	defer func() { _ = application.Close() }()

	application.Logger.Info("开始执行数据库清理")

	// Check and drop 'we_chat_id' column from 'users' table if it exists
	if database.HasColumn(application.DB, &models.User{}, "we_chat_id") {
		application.Logger.Info("发现旧字段 we_chat_id，准备删除")
		if err := database.DropColumn(application.DB, &models.User{}, "we_chat_id"); err != nil {
			application.Logger.Fatal("删除旧字段失败", zap.Error(err))
		}
		application.Logger.Info("旧字段删除完成")
	} else {
		application.Logger.Info("未发现旧字段 we_chat_id")
	}

	application.Logger.Info("数据库清理完成")
}
