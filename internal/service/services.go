package service

import (
	"go.uber.org/zap"

	"vocalin-backend/internal/auth"
)

// Services 统一管理业务服务，便于在路由层注入。
type Services struct {
	Auth    *AuthService
	Group   *GroupService
	Home    *HomeService
	Record  *RecordService
	Profile *ProfileService
}

func NewServices(store Store, tokenManager *auth.TokenManager, logger *zap.Logger) *Services {
	return &Services{
		Auth:    NewAuthService(store, tokenManager, logger),
		Group:   NewGroupService(store, logger),
		Home:    NewHomeService(store, logger),
		Record:  NewRecordService(store, logger),
		Profile: NewProfileService(store, logger),
	}
}
