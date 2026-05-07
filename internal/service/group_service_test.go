package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"gorm.io/gorm"

	"vocalin-backend/internal/models"
)

func TestGroupServiceGetGroupInfoIncludesMemberRoles(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-group-info", Nickname: "owner-group-info", StatusUpdatedAt: time.Now()}
	member := &models.User{WeChatID: "member-group-info", Nickname: "member-group-info", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-info")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group.ID); err != nil {
		t.Fatalf("add member to group: %v", err)
	}

	result, err := svc.GetGroupInfo(ctx, owner.ID)
	if err != nil {
		t.Fatalf("get group info: %v", err)
	}
	if result.MyRole != GroupRoleOwner {
		t.Fatalf("expected my role %q, got %q", GroupRoleOwner, result.MyRole)
	}
	if len(result.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(result.Members))
	}

	sort.Slice(result.Members, func(i, j int) bool {
		return result.Members[i].ID < result.Members[j].ID
	})

	if result.Members[0].ID != owner.ID || result.Members[0].GroupRole != GroupRoleOwner {
		t.Fatalf("expected owner member role %q, got %+v", GroupRoleOwner, result.Members[0])
	}
	if result.Members[1].ID != member.ID || result.Members[1].GroupRole != GroupRoleMember {
		t.Fatalf("expected member role %q, got %+v", GroupRoleMember, result.Members[1])
	}
}

func TestGroupServiceLeaveGroupFallsBackToRemainingCurrentGroup(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-leave", Nickname: "owner-leave", StatusUpdatedAt: time.Now()}
	member := &models.User{WeChatID: "member-leave", Nickname: "member-leave", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	group1, err := svc.CreateGroup(ctx, owner.ID, "g1")
	if err != nil {
		t.Fatalf("create group1: %v", err)
	}
	group2, err := svc.CreateGroup(ctx, owner.ID, "g2")
	if err != nil {
		t.Fatalf("create group2: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group1.ID); err != nil {
		t.Fatalf("add member to group1: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group2.ID); err != nil {
		t.Fatalf("add member to group2: %v", err)
	}
	if _, err := svc.SwitchCurrentGroup(ctx, member.ID, group1.ID); err != nil {
		t.Fatalf("switch current group: %v", err)
	}

	result, err := svc.LeaveGroup(ctx, member.ID, group1.ID)
	if err != nil {
		t.Fatalf("leave group: %v", err)
	}
	if result.CurrentGroupID == nil || *result.CurrentGroupID != group2.ID {
		t.Fatalf("expected fallback current group %d, got %v", group2.ID, result.CurrentGroupID)
	}
	if result.FallbackGroup == nil || result.FallbackGroup.ID != group2.ID {
		t.Fatalf("expected fallback group %d, got %+v", group2.ID, result.FallbackGroup)
	}

	reloaded, err := store.GetUserByID(ctx, member.ID)
	if err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if reloaded.CurrentGroupID == nil || *reloaded.CurrentGroupID != group2.ID {
		t.Fatalf("expected member current group %d, got %v", group2.ID, reloaded.CurrentGroupID)
	}
	if _, err := store.GetGroupMember(ctx, group1.ID, member.ID); err == nil {
		t.Fatal("expected membership in left group to be removed")
	}
}

func TestGroupServiceTransferOwnershipAndDisbandGroup(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	homeSvc := NewHomeService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-disband", Nickname: "owner-disband", StatusUpdatedAt: time.Now()}
	member := &models.User{WeChatID: "member-disband", Nickname: "member-disband", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}

	group1, err := svc.CreateGroup(ctx, owner.ID, "group-to-disband")
	if err != nil {
		t.Fatalf("create group1: %v", err)
	}
	group2, err := svc.CreateGroup(ctx, member.ID, "member-backup")
	if err != nil {
		t.Fatalf("create group2: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group1.ID); err != nil {
		t.Fatalf("add member to group1: %v", err)
	}
	if _, err := svc.SwitchCurrentGroup(ctx, member.ID, group1.ID); err != nil {
		t.Fatalf("switch member current group: %v", err)
	}

	if err := svc.TransferOwnership(ctx, owner.ID, group1.ID, member.ID); err != nil {
		t.Fatalf("transfer ownership: %v", err)
	}
	messages, err := homeSvc.ListMessages(ctx, member.ID)
	if err != nil {
		t.Fatalf("list transfer messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 transfer message, got %+v", messages)
	}
	if err := svc.ReviewOwnershipTransfer(ctx, member.ID, group1.ID, "approve"); err != nil {
		t.Fatalf("approve transfer message: %v", err)
	}
	ownerMembership, err := store.GetGroupMember(ctx, group1.ID, owner.ID)
	if err != nil {
		t.Fatalf("reload owner membership: %v", err)
	}
	if ownerMembership.Role != GroupRoleMember {
		t.Fatalf("expected old owner role %q, got %q", GroupRoleMember, ownerMembership.Role)
	}
	memberMembership, err := store.GetGroupMember(ctx, group1.ID, member.ID)
	if err != nil {
		t.Fatalf("reload member membership: %v", err)
	}
	if memberMembership.Role != GroupRoleOwner {
		t.Fatalf("expected new owner role %q, got %q", GroupRoleOwner, memberMembership.Role)
	}

	result, err := svc.DisbandGroup(ctx, member.ID, group1.ID)
	if err != nil {
		t.Fatalf("disband group: %v", err)
	}
	if result.CurrentGroupID == nil || *result.CurrentGroupID != group2.ID {
		t.Fatalf("expected member fallback current group %d, got %v", group2.ID, result.CurrentGroupID)
	}
	if _, err := store.GetGroupWithMembers(ctx, group1.ID); err == nil {
		t.Fatal("expected disbanded group to be unavailable")
	}
	memberReloaded, err := store.GetUserByID(ctx, member.ID)
	if err != nil {
		t.Fatalf("reload member: %v", err)
	}
	if memberReloaded.CurrentGroupID == nil || *memberReloaded.CurrentGroupID != group2.ID {
		t.Fatalf("expected member current group %d, got %v", group2.ID, memberReloaded.CurrentGroupID)
	}
	ownerReloaded, err := store.GetUserByID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("reload owner: %v", err)
	}
	if ownerReloaded.CurrentGroupID != nil {
		t.Fatalf("expected old owner to have no remaining current group, got %v", ownerReloaded.CurrentGroupID)
	}
}

func TestGroupServiceRemoveMemberUpdatesTargetFallbackGroup(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-remove", Nickname: "owner-remove", StatusUpdatedAt: time.Now()}
	target := &models.User{WeChatID: "target-remove", Nickname: "target-remove", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	group1, err := svc.CreateGroup(ctx, owner.ID, "group-remove")
	if err != nil {
		t.Fatalf("create group1: %v", err)
	}
	group2, err := svc.CreateGroup(ctx, target.ID, "group-keep")
	if err != nil {
		t.Fatalf("create group2: %v", err)
	}
	if err := store.AddUserToGroup(ctx, target, group1.ID); err != nil {
		t.Fatalf("add target to group1: %v", err)
	}
	if _, err := svc.SwitchCurrentGroup(ctx, target.ID, group1.ID); err != nil {
		t.Fatalf("switch target current group: %v", err)
	}

	if err := svc.RemoveMember(ctx, owner.ID, group1.ID, target.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	if _, err := store.GetGroupMember(ctx, group1.ID, target.ID); err == nil {
		t.Fatal("expected removed member to be absent from group")
	}
	targetReloaded, err := store.GetUserByID(ctx, target.ID)
	if err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if targetReloaded.CurrentGroupID == nil || *targetReloaded.CurrentGroupID != group2.ID {
		t.Fatalf("expected target fallback current group %d, got %v", group2.ID, targetReloaded.CurrentGroupID)
	}
	ownerMembership, err := store.GetGroupMember(ctx, group1.ID, owner.ID)
	if err != nil {
		t.Fatalf("reload owner membership: %v", err)
	}
	if ownerMembership.Role != GroupRoleOwner {
		t.Fatalf("expected owner role %q, got %q", GroupRoleOwner, ownerMembership.Role)
	}
	if group2.CreatorID != target.ID {
		t.Fatalf("expected group2 creator %d, got %d", target.ID, group2.CreatorID)
	}
	if targetReloaded.CurrentGroupID == nil || *targetReloaded.CurrentGroupID == group1.ID {
		t.Fatalf("expected target current group to move away from removed group, got %v", targetReloaded.CurrentGroupID)
	}
}

func TestGroupServiceRemoveMemberRejectsSelfRemoval(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-self-remove", Nickname: "owner-self-remove", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-self-remove")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	err = svc.RemoveMember(ctx, owner.ID, group.ID, owner.ID)
	if err != ErrCannotRemoveSelf {
		t.Fatalf("expected ErrCannotRemoveSelf, got %v", err)
	}
}

func TestGroupServiceRemoveMemberRequiresOwner(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-only-remove", Nickname: "owner-only-remove", StatusUpdatedAt: time.Now()}
	member := &models.User{WeChatID: "member-only-remove", Nickname: "member-only-remove", StatusUpdatedAt: time.Now()}
	target := &models.User{WeChatID: "target-only-remove", Nickname: "target-only-remove", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := store.CreateUser(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-owner-only-remove")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, member, group.ID); err != nil {
		t.Fatalf("add member to group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, target, group.ID); err != nil {
		t.Fatalf("add target to group: %v", err)
	}

	err = svc.RemoveMember(ctx, member.ID, group.ID, target.ID)
	if err != ErrGroupOwnerOnly {
		t.Fatalf("expected ErrGroupOwnerOnly, got %v", err)
	}

	if _, err := store.GetGroupMember(ctx, group.ID, target.ID); err != nil {
		t.Fatalf("expected target membership to remain, got %v", err)
	}
}

func TestGroupServiceJoinGroupCreatesPendingRequest(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	homeSvc := NewHomeService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-pending-join", Nickname: "owner-pending-join", StatusUpdatedAt: time.Now()}
	applicant := &models.User{WeChatID: "applicant-pending-join", Nickname: "applicant-pending-join", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, applicant); err != nil {
		t.Fatalf("create applicant: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-pending-join")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	joinedGroup, err := svc.JoinGroup(ctx, applicant.ID, group.InviteCode)
	if err != nil {
		t.Fatalf("join group: %v", err)
	}
	if joinedGroup.ID != group.ID {
		t.Fatalf("expected response group %d, got %d", group.ID, joinedGroup.ID)
	}
	if _, err := store.GetGroupMember(ctx, group.ID, applicant.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected applicant membership to stay pending, got %v", err)
	}

	listResult, err := svc.ListGroups(ctx, applicant.ID)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(listResult.Groups) != 0 {
		t.Fatalf("expected no active groups for applicant, got %+v", listResult.Groups)
	}
	if len(listResult.PendingRequests) != 1 {
		t.Fatalf("expected 1 pending request, got %+v", listResult.PendingRequests)
	}
	if listResult.PendingRequests[0].Type != models.GroupRequestTypeJoin {
		t.Fatalf("expected pending join request, got %+v", listResult.PendingRequests[0])
	}
	if listResult.PendingRequests[0].GroupID != group.ID {
		t.Fatalf("expected pending request for group %d, got %+v", group.ID, listResult.PendingRequests[0])
	}

	dashboard, err := homeSvc.GetDashboard(ctx, owner.ID)
	if err != nil {
		t.Fatalf("get owner dashboard: %v", err)
	}
	if dashboard.PendingMessageCount != 1 {
		t.Fatalf("expected owner pending message count 1, got %d", dashboard.PendingMessageCount)
	}

	messages, err := homeSvc.ListMessages(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list owner messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 owner message, got %+v", messages)
	}
	if messages[0].Type != models.GroupRequestTypeJoin {
		t.Fatalf("expected join request message, got %+v", messages[0])
	}

	if err := svc.ReviewJoinRequest(ctx, owner.ID, group.ID, messages[0].ID, "approve"); err != nil {
		t.Fatalf("approve join request: %v", err)
	}
	membership, err := store.GetGroupMember(ctx, group.ID, applicant.ID)
	if err != nil {
		t.Fatalf("load applicant membership: %v", err)
	}
	if membership.Role != GroupRoleMember {
		t.Fatalf("expected approved applicant role %q, got %q", GroupRoleMember, membership.Role)
	}
}

func TestGroupServiceTransferOwnershipCreatesPendingRequest(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	homeSvc := NewHomeService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-pending-transfer", Nickname: "owner-pending-transfer", StatusUpdatedAt: time.Now()}
	target := &models.User{WeChatID: "target-pending-transfer", Nickname: "target-pending-transfer", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-pending-transfer")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, target, group.ID); err != nil {
		t.Fatalf("add target to group: %v", err)
	}

	if err := svc.TransferOwnership(ctx, owner.ID, group.ID, target.ID); err != nil {
		t.Fatalf("transfer ownership request: %v", err)
	}

	ownerMembership, err := store.GetGroupMember(ctx, group.ID, owner.ID)
	if err != nil {
		t.Fatalf("load owner membership: %v", err)
	}
	if ownerMembership.Role != GroupRoleOwner {
		t.Fatalf("expected owner role to remain %q before approval, got %q", GroupRoleOwner, ownerMembership.Role)
	}
	targetMembership, err := store.GetGroupMember(ctx, group.ID, target.ID)
	if err != nil {
		t.Fatalf("load target membership: %v", err)
	}
	if targetMembership.Role != GroupRoleMember {
		t.Fatalf("expected target role to remain %q before approval, got %q", GroupRoleMember, targetMembership.Role)
	}

	ownerGroup, err := svc.GetGroupInfo(ctx, owner.ID)
	if err != nil {
		t.Fatalf("get owner group info: %v", err)
	}
	if !ownerGroup.PendingOwnershipTransfer {
		t.Fatalf("expected group to expose pending ownership transfer, got %+v", ownerGroup)
	}
	if ownerGroup.PendingOwnershipTransferToUserID == nil || *ownerGroup.PendingOwnershipTransferToUserID != target.ID {
		t.Fatalf("expected pending transfer target %d, got %+v", target.ID, ownerGroup.PendingOwnershipTransferToUserID)
	}

	dashboard, err := homeSvc.GetDashboard(ctx, target.ID)
	if err != nil {
		t.Fatalf("get target dashboard: %v", err)
	}
	if dashboard.PendingMessageCount != 1 {
		t.Fatalf("expected target pending message count 1, got %d", dashboard.PendingMessageCount)
	}

	messages, err := homeSvc.ListMessages(ctx, target.ID)
	if err != nil {
		t.Fatalf("list target messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 target message, got %+v", messages)
	}
	if messages[0].Type != models.GroupRequestTypeOwnershipTransfer {
		t.Fatalf("expected ownership transfer message, got %+v", messages[0])
	}

	if err := svc.ReviewOwnershipTransfer(ctx, target.ID, group.ID, "approve"); err != nil {
		t.Fatalf("approve transfer request: %v", err)
	}

	ownerMembership, err = store.GetGroupMember(ctx, group.ID, owner.ID)
	if err != nil {
		t.Fatalf("reload owner membership: %v", err)
	}
	if ownerMembership.Role != GroupRoleMember {
		t.Fatalf("expected old owner role %q after approval, got %q", GroupRoleMember, ownerMembership.Role)
	}
	targetMembership, err = store.GetGroupMember(ctx, group.ID, target.ID)
	if err != nil {
		t.Fatalf("reload target membership: %v", err)
	}
	if targetMembership.Role != GroupRoleOwner {
		t.Fatalf("expected new owner role %q after approval, got %q", GroupRoleOwner, targetMembership.Role)
	}
}

func TestGroupServiceRejectJoinRequestLeavesApplicantOutOfGroup(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	homeSvc := NewHomeService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-reject-join", Nickname: "owner-reject-join", StatusUpdatedAt: time.Now()}
	applicant := &models.User{WeChatID: "applicant-reject-join", Nickname: "applicant-reject-join", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, applicant); err != nil {
		t.Fatalf("create applicant: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-reject-join")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := svc.JoinGroup(ctx, applicant.ID, group.InviteCode); err != nil {
		t.Fatalf("join group: %v", err)
	}

	messages, err := homeSvc.ListMessages(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list owner messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 owner message, got %+v", messages)
	}

	if err := svc.ReviewJoinRequest(ctx, owner.ID, group.ID, messages[0].ID, "reject"); err != nil {
		t.Fatalf("reject join request: %v", err)
	}
	if _, err := store.GetGroupMember(ctx, group.ID, applicant.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected applicant membership to remain absent after rejection, got %v", err)
	}

	messages, err = homeSvc.ListMessages(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list owner messages after rejection: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected no pending owner messages after rejection, got %+v", messages)
	}
}

func TestGroupServiceRejectTransferRequestLeavesRolesUnchanged(t *testing.T) {
	store := newTestStore(t)
	svc := NewGroupService(store, newTestLogger())
	homeSvc := NewHomeService(store, newTestLogger())
	ctx := context.Background()

	owner := &models.User{WeChatID: "owner-reject-transfer", Nickname: "owner-reject-transfer", StatusUpdatedAt: time.Now()}
	target := &models.User{WeChatID: "target-reject-transfer", Nickname: "target-reject-transfer", StatusUpdatedAt: time.Now()}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := store.CreateUser(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	group, err := svc.CreateGroup(ctx, owner.ID, "group-reject-transfer")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := store.AddUserToGroup(ctx, target, group.ID); err != nil {
		t.Fatalf("add target to group: %v", err)
	}
	if err := svc.TransferOwnership(ctx, owner.ID, group.ID, target.ID); err != nil {
		t.Fatalf("transfer ownership request: %v", err)
	}

	messages, err := homeSvc.ListMessages(ctx, target.ID)
	if err != nil {
		t.Fatalf("list target messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 target message, got %+v", messages)
	}

	if err := svc.ReviewOwnershipTransfer(ctx, target.ID, group.ID, "reject"); err != nil {
		t.Fatalf("reject transfer request: %v", err)
	}

	ownerMembership, err := store.GetGroupMember(ctx, group.ID, owner.ID)
	if err != nil {
		t.Fatalf("reload owner membership: %v", err)
	}
	if ownerMembership.Role != GroupRoleOwner {
		t.Fatalf("expected owner role %q after rejection, got %q", GroupRoleOwner, ownerMembership.Role)
	}
	targetMembership, err := store.GetGroupMember(ctx, group.ID, target.ID)
	if err != nil {
		t.Fatalf("reload target membership: %v", err)
	}
	if targetMembership.Role != GroupRoleMember {
		t.Fatalf("expected target role %q after rejection, got %q", GroupRoleMember, targetMembership.Role)
	}

	messages, err = homeSvc.ListMessages(ctx, target.ID)
	if err != nil {
		t.Fatalf("list target messages after rejection: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected no pending target messages after rejection, got %+v", messages)
	}
}
