package service

import "errors"

var (
	ErrUnauthorized            = errors.New("未授权")
	ErrInvalidCredentials      = errors.New("昵称或密码错误")
	ErrNicknameAlreadyExists   = errors.New("昵称已被使用")
	ErrPhoneAlreadyExists      = errors.New("手机号已被使用")
	ErrPasswordMismatch        = errors.New("两次输入的密码不一致")
	ErrUserAlreadyInGroup      = errors.New("用户已加入空间")
	ErrUserNotInGroup          = errors.New("用户尚未加入空间")
	ErrGroupNotFound           = errors.New("空间不存在")
	ErrInvalidInviteCode       = errors.New("邀请码无效")
	ErrInvalidRefreshToken     = errors.New("刷新令牌无效")
	ErrRefreshTokenRevoked     = errors.New("刷新令牌已失效")
	ErrTimedNoteRequiresShowAt = errors.New("定时便签必须提供展示时间")
	ErrWishlistItemNotFound    = errors.New("愿望条目不存在")
	ErrForbidden               = errors.New("无权访问该资源")
)
