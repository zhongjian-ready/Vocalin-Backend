package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"
)

type GroupHandler struct {
	groupService *service.GroupService
}

type GroupResponse = models.Group
type GroupListResponse = service.GroupListResult
type GroupSwitchResponse = service.GroupSwitchResult
type GroupFallbackResponse = service.GroupFallbackResult

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required,min=2,max=50"`
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" binding:"required,invite_code"`
}

type SwitchGroupRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
}

type TransferGroupOwnershipRequest struct {
	TargetUserID uint `json:"target_user_id" binding:"required"`
}

type ReviewGroupRequestActionRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject"`
}

func NewGroupHandler(groupService *service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// CreateGroup godoc
// @Summary 创建空间
// @Description 创建一个新的亲密空间，并自动将当前用户加入空间
// @Tags Group
// @Accept json
// @Produce json
// @Param request body CreateGroupRequest true "Create Group Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/create [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.groupService.CreateGroup(c.Request.Context(), currentUserID(c), req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "空间创建成功", group)
}

// JoinGroup godoc
// @Summary 加入空间
// @Description 使用邀请码发起加入申请，待群组管理员同意后才会正式加入空间
// @Tags Group
// @Accept json
// @Produce json
// @Param request body JoinGroupRequest true "Join Group Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/join [post]
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	var req JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	group, err := h.groupService.JoinGroup(c.Request.Context(), currentUserID(c), req.InviteCode)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "已发起申请", group)
}

// ListGroups godoc
// @Summary 获取我的群组列表
// @Description 获取当前登录用户已加入的全部群组、当前激活群组，以及我发起但仍在处理中申请/移交记录
// @Tags Group
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupListResponse}
// @Router /groups [get]
func (h *GroupHandler) ListGroups(c *gin.Context) {
	result, err := h.groupService.ListGroups(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取群组列表成功", result)
}

// GetGroupInfo godoc
// @Summary 获取当前空间信息
// @Description 获取当前激活空间及成员信息；若当前用户已发起管理员移交，会返回移交中的状态字段
// @Tags Group
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupResponse}
// @Router /groups/current [get]
func (h *GroupHandler) GetGroupInfo(c *gin.Context) {
	group, err := h.groupService.GetGroupInfo(c.Request.Context(), currentUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "获取空间成功", group)
}

// SwitchCurrentGroup godoc
// @Summary 切换当前空间
// @Description 将当前激活空间切换为已正式加入的其他空间，申请中的空间不能切换
// @Tags Group
// @Accept json
// @Produce json
// @Param request body SwitchGroupRequest true "Switch Group Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupSwitchResponse}
// @Router /groups/current [put]
func (h *GroupHandler) SwitchCurrentGroup(c *gin.Context) {
	var req SwitchGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	result, err := h.groupService.SwitchCurrentGroup(c.Request.Context(), currentUserID(c), req.GroupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "切换当前空间成功", result)
}

// LeaveGroup godoc
// @Summary 退出群组
// @Description 退出指定群组；若退出的是当前激活群组，将自动选择其余群组中的第一个作为当前空间
// @Tags Group
// @Produce json
// @Param groupId path int true "Group ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupFallbackResponse}
// @Router /groups/{groupId}/members/me [delete]
func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	result, err := h.groupService.LeaveGroup(c.Request.Context(), currentUserID(c), groupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "退出群组成功", result)
}

// RemoveGroupMember godoc
// @Summary 移除群组成员
// @Description 当前群组管理员可将指定成员移出群组
// @Tags Group
// @Produce json
// @Param groupId path int true "Group ID"
// @Param userId path int true "Target User ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /groups/{groupId}/members/{userId} [delete]
func (h *GroupHandler) RemoveGroupMember(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	targetUserID, ok := userIDFromParam(c, "userId")
	if !ok {
		return
	}

	if err := h.groupService.RemoveMember(c.Request.Context(), currentUserID(c), groupID, targetUserID); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "移除群组成员成功", nil)
}

// TransferGroupOwnership godoc
// @Summary 转让群组管理权
// @Description 当前群组管理员可向同群组其他成员发起管理权移交申请，待对方同意后才正式生效
// @Tags Group
// @Accept json
// @Produce json
// @Param groupId path int true "Group ID"
// @Param request body TransferGroupOwnershipRequest true "Transfer Group Ownership Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /groups/{groupId}/owner [put]
func (h *GroupHandler) TransferGroupOwnership(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	var req TransferGroupOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	if err := h.groupService.TransferOwnership(c.Request.Context(), currentUserID(c), groupID, req.TargetUserID); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "已发起移交", nil)
}

// ReviewJoinRequest godoc
// @Summary 审批加入申请
// @Description 群组管理员对指定加入申请执行同意或拒绝，action 仅支持 approve 或 reject
// @Tags Group
// @Accept json
// @Produce json
// @Param groupId path int true "Group ID"
// @Param requestId path int true "Request ID"
// @Param request body ReviewGroupRequestActionRequest true "Review Join Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /groups/{groupId}/join-requests/{requestId}/review [post]
func (h *GroupHandler) ReviewJoinRequest(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	requestID, ok := uintFromParam(c, "requestId", "非法的申请 ID")
	if !ok {
		return
	}

	var req ReviewGroupRequestActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	if err := h.groupService.ReviewJoinRequest(c.Request.Context(), currentUserID(c), groupID, requestID, req.Action); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "处理加入申请成功", nil)
}

// ReviewOwnershipTransfer godoc
// @Summary 审批管理权移交
// @Description 被移交方对当前群组待处理的管理权移交执行同意或拒绝，action 仅支持 approve 或 reject
// @Tags Group
// @Accept json
// @Produce json
// @Param groupId path int true "Group ID"
// @Param request body ReviewGroupRequestActionRequest true "Review Ownership Transfer Request"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse
// @Router /groups/{groupId}/owner/review [post]
func (h *GroupHandler) ReviewOwnershipTransfer(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	var req ReviewGroupRequestActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	if err := h.groupService.ReviewOwnershipTransfer(c.Request.Context(), currentUserID(c), groupID, req.Action); err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "处理管理权移交成功", nil)
}

// DisbandGroup godoc
// @Summary 解散群组
// @Description 当前群组管理员可解散群组，所有成员会失去该群组访问权限
// @Tags Group
// @Produce json
// @Param groupId path int true "Group ID"
// @Security BearerAuth
// @Success 200 {object} response.APIResponse{data=GroupFallbackResponse}
// @Router /groups/{groupId} [delete]
func (h *GroupHandler) DisbandGroup(c *gin.Context) {
	groupID, ok := groupIDFromParam(c)
	if !ok {
		return
	}

	result, err := h.groupService.DisbandGroup(c.Request.Context(), currentUserID(c), groupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "解散群组成功", result)
}

func groupIDFromParam(c *gin.Context) (uint, bool) {
	return uintFromParam(c, "groupId", "非法的群组 ID")
}

func userIDFromParam(c *gin.Context, param string) (uint, bool) {
	return uintFromParam(c, param, "非法的用户 ID")
}

func uintFromParam(c *gin.Context, param string, message string) (uint, bool) {
	groupID, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || groupID == 0 {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
		return 0, false
	}
	return uint(groupID), true
}
