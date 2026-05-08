package service

import "errors"

const (
	GroupRoleOwner  = "owner"
	GroupRoleMember = "member"
)

var (
	ErrUnauthorized            = errors.New("未授权")
	ErrInvalidCredentials      = errors.New("昵称或密码错误")
	ErrNicknameRequired        = errors.New("昵称不能为空")
	ErrNicknameAlreadyExists   = errors.New("昵称已被使用")
	ErrPhoneAlreadyExists      = errors.New("手机号已被使用")
	ErrPasswordMismatch        = errors.New("两次输入的密码不一致")
	ErrUserAlreadyInGroup      = errors.New("用户已加入空间")
	ErrUserNotInGroup          = errors.New("用户尚未加入空间")
	ErrGroupJoinRequestPending = errors.New("已发起申请，请等待管理员同意")
	ErrGroupNotFound           = errors.New("空间不存在")
	ErrInvalidInviteCode       = errors.New("邀请码无效")
	ErrGroupOwnershipTransfer  = errors.New("当前群组管理员退出前需先转让管理权")
	ErrGroupTransferPending    = errors.New("已发起移交，请等待对方同意")
	ErrGroupMemberLimitReached = errors.New("群组成员已达上限，最多只能有24个成员")
	ErrGroupOwnerOnly          = errors.New("仅群组管理员可执行该操作")
	ErrGroupMemberNotFound     = errors.New("目标成员不存在")
	ErrGroupRequestNotFound    = errors.New("消息不存在")
	ErrGroupRequestHandled     = errors.New("消息已处理")
	ErrCannotRemoveSelf        = errors.New("不能移除自己，请使用退出空间或先转让管理权")
	ErrCannotRemoveGroupOwner  = errors.New("不能移除群组管理员")
	ErrCannotTransferToSelf    = errors.New("不能将群组管理权转让给自己")
	ErrInvalidRefreshToken     = errors.New("刷新令牌无效")
	ErrRefreshTokenRevoked     = errors.New("刷新令牌已失效")
	ErrTimedNoteRequiresShowAt = errors.New("定时便签必须提供展示时间")
	ErrPhotoNotFound           = errors.New("照片不存在")
	ErrNoteNotFound            = errors.New("便签不存在")
	ErrWishlistItemNotFound    = errors.New("愿望条目不存在")
	ErrForbidden               = errors.New("无权访问该资源")
)
