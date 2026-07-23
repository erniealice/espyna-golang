package subscription_group

// Unit tests for the referential delete guard: a subscription_group must NOT be
// deletable while ACTIVE dependents (roster members / class staff / access
// grants) reference its id. Every dependent repo is mock-backed; the List calls
// return canned rows and the guard re-filters in memory (active + group + ws).
//
// The mocks ignore the List filter (the real postgres adapter applies it +
// auto-scopes active=true); returning the full canned set exercises the
// use-case in-memory scoping directly, which is the layer under test.

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	appcontext "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	memberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	sgppspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	sgwupb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

// ----- mocks ---------------------------------------------------------------

type mockSGDeleteRepo struct {
	pb.UnimplementedSubscriptionGroupDomainServiceServer
	deleteCalls int
}

func (m *mockSGDeleteRepo) DeleteSubscriptionGroup(_ context.Context, _ *pb.DeleteSubscriptionGroupRequest) (*pb.DeleteSubscriptionGroupResponse, error) {
	m.deleteCalls++
	return &pb.DeleteSubscriptionGroupResponse{Success: true}, nil
}

type mockMemberRepo struct {
	memberpb.UnimplementedSubscriptionGroupMemberDomainServiceServer
	rows []*memberpb.SubscriptionGroupMember
}

func (m *mockMemberRepo) ListSubscriptionGroupMembers(_ context.Context, _ *memberpb.ListSubscriptionGroupMembersRequest) (*memberpb.ListSubscriptionGroupMembersResponse, error) {
	return &memberpb.ListSubscriptionGroupMembersResponse{Success: true, Data: m.rows}, nil
}

type mockStaffRepo struct {
	sgppspb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	rows []*sgppspb.SubscriptionGroupProductPlanStaff
}

func (m *mockStaffRepo) ListSubscriptionGroupProductPlanStaffs(_ context.Context, _ *sgppspb.ListSubscriptionGroupProductPlanStaffsRequest) (*sgppspb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	return &sgppspb.ListSubscriptionGroupProductPlanStaffsResponse{Success: true, Data: m.rows}, nil
}

type mockGrantRepo struct {
	sgwupb.UnimplementedSubscriptionGroupWorkspaceUserDomainServiceServer
	rows []*sgwupb.SubscriptionGroupWorkspaceUser
}

func (m *mockGrantRepo) ListSubscriptionGroupWorkspaceUsers(_ context.Context, _ *sgwupb.ListSubscriptionGroupWorkspaceUsersRequest) (*sgwupb.ListSubscriptionGroupWorkspaceUsersResponse, error) {
	return &sgwupb.ListSubscriptionGroupWorkspaceUsersResponse{Success: true, Data: m.rows}, nil
}

// ----- helpers -------------------------------------------------------------

const (
	sgID = "sg-1"
	wsA  = "ws-1"
	wsB  = "ws-2"
)

func member(active bool, ws string) *memberpb.SubscriptionGroupMember {
	return &memberpb.SubscriptionGroupMember{Id: "m-" + ws, SubscriptionGroupId: sgID, Active: active, WorkspaceId: ws}
}

func staffEdge(active bool, ws string) *sgppspb.SubscriptionGroupProductPlanStaff {
	return &sgppspb.SubscriptionGroupProductPlanStaff{Id: "s-" + ws, SubscriptionGroupId: sgID, Active: active, WorkspaceId: ws}
}

func grant(active bool, ws string) *sgwupb.SubscriptionGroupWorkspaceUser {
	return &sgwupb.SubscriptionGroupWorkspaceUser{Id: "g-" + ws, SubscriptionGroupId: sgID, Active: active, WorkspaceId: ws}
}

func newDeleteUC(sg *mockSGDeleteRepo, mem *mockMemberRepo, st *mockStaffRepo, gr *mockGrantRepo) *DeleteSubscriptionGroupUseCase {
	return NewDeleteSubscriptionGroupUseCase(
		DeleteSubscriptionGroupRepositories{
			SubscriptionGroup: sg,
			Member:            mem,
			TeachingStaff:     st,
			AccessGrant:       gr,
		},
		DeleteSubscriptionGroupServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			IDGenerator:      ports.NewNoOpIDGenerator(),
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		},
	)
}

func deleteReq() *pb.DeleteSubscriptionGroupRequest {
	return &pb.DeleteSubscriptionGroupRequest{Data: &pb.SubscriptionGroup{Id: sgID}}
}

// ----- tests ---------------------------------------------------------------

func TestDeleteSubscriptionGroup_BlockedByMembers(t *testing.T) {
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg,
		&mockMemberRepo{rows: []*memberpb.SubscriptionGroupMember{member(true, ""), member(true, "")}},
		&mockStaffRepo{}, &mockGrantRepo{})

	_, err := uc.Execute(context.Background(), deleteReq())
	if err == nil {
		t.Fatalf("expected delete to be blocked by active members")
	}
	if !strings.Contains(err.Error(), "2 members") {
		t.Errorf("message should enumerate 2 members, got: %q", err.Error())
	}
	if sg.deleteCalls != 0 {
		t.Errorf("delete must NOT be called when blocked, got %d calls", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_BlockedByTeachingStaff(t *testing.T) {
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg, &mockMemberRepo{},
		&mockStaffRepo{rows: []*sgppspb.SubscriptionGroupProductPlanStaff{staffEdge(true, "")}},
		&mockGrantRepo{})

	_, err := uc.Execute(context.Background(), deleteReq())
	if err == nil {
		t.Fatalf("expected delete to be blocked by active teaching staff")
	}
	if !strings.Contains(err.Error(), "1 staff assignment") {
		t.Errorf("message should enumerate 1 staff assignment, got: %q", err.Error())
	}
	if sg.deleteCalls != 0 {
		t.Errorf("delete must NOT be called when blocked, got %d calls", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_BlockedByAccessGrants(t *testing.T) {
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg, &mockMemberRepo{}, &mockStaffRepo{},
		&mockGrantRepo{rows: []*sgwupb.SubscriptionGroupWorkspaceUser{grant(true, ""), grant(true, ""), grant(true, "")}})

	_, err := uc.Execute(context.Background(), deleteReq())
	if err == nil {
		t.Fatalf("expected delete to be blocked by active access grants")
	}
	if !strings.Contains(err.Error(), "3 access grants") {
		t.Errorf("message should enumerate 3 access grants, got: %q", err.Error())
	}
	if sg.deleteCalls != 0 {
		t.Errorf("delete must NOT be called when blocked, got %d calls", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_BlockedByMultiple_EnumeratesAll(t *testing.T) {
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg,
		&mockMemberRepo{rows: []*memberpb.SubscriptionGroupMember{member(true, ""), member(true, ""), member(true, "")}},
		&mockStaffRepo{rows: []*sgppspb.SubscriptionGroupProductPlanStaff{staffEdge(true, ""), staffEdge(true, "")}},
		&mockGrantRepo{rows: []*sgwupb.SubscriptionGroupWorkspaceUser{grant(true, "")}})

	_, err := uc.Execute(context.Background(), deleteReq())
	if err == nil {
		t.Fatalf("expected delete to be blocked")
	}
	msg := err.Error()
	for _, want := range []string{"3 members", "2 staff assignments", "1 access grant"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should enumerate %q, got: %q", want, msg)
		}
	}
	if sg.deleteCalls != 0 {
		t.Errorf("delete must NOT be called when blocked, got %d calls", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_ZeroDependents_Deletes(t *testing.T) {
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg,
		// Inactive rows must NOT block — only ACTIVE dependents count.
		&mockMemberRepo{rows: []*memberpb.SubscriptionGroupMember{member(false, "")}},
		&mockStaffRepo{rows: []*sgppspb.SubscriptionGroupProductPlanStaff{staffEdge(false, "")}},
		&mockGrantRepo{rows: []*sgwupb.SubscriptionGroupWorkspaceUser{grant(false, "")}})

	resp, err := uc.Execute(context.Background(), deleteReq())
	if err != nil {
		t.Fatalf("delete with zero active dependents should succeed, got: %v", err)
	}
	if resp == nil || !resp.GetSuccess() {
		t.Errorf("expected a successful delete response")
	}
	if sg.deleteCalls != 1 {
		t.Errorf("delete should be called exactly once, got %d", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_WorkspaceScoped_OtherWorkspaceDoesNotBlock(t *testing.T) {
	// Caller is bound to workspace A; all live dependents belong to workspace B.
	// The guard must scope to A and let the delete proceed.
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg,
		&mockMemberRepo{rows: []*memberpb.SubscriptionGroupMember{member(true, wsB), member(true, wsB)}},
		&mockStaffRepo{rows: []*sgppspb.SubscriptionGroupProductPlanStaff{staffEdge(true, wsB)}},
		&mockGrantRepo{rows: []*sgwupb.SubscriptionGroupWorkspaceUser{grant(true, wsB)}})

	ctx := appcontext.WithWorkspaceID(context.Background(), wsA)
	resp, err := uc.Execute(ctx, deleteReq())
	if err != nil {
		t.Fatalf("dependents in ANOTHER workspace must not block; got: %v", err)
	}
	if resp == nil || !resp.GetSuccess() {
		t.Errorf("expected a successful delete response")
	}
	if sg.deleteCalls != 1 {
		t.Errorf("delete should be called exactly once, got %d", sg.deleteCalls)
	}
}

func TestDeleteSubscriptionGroup_WorkspaceScoped_SameWorkspaceBlocks(t *testing.T) {
	// Same setup but the caller is bound to workspace B (where the dependents
	// live) — the guard must block.
	sg := &mockSGDeleteRepo{}
	uc := newDeleteUC(sg,
		&mockMemberRepo{rows: []*memberpb.SubscriptionGroupMember{member(true, wsB)}},
		&mockStaffRepo{}, &mockGrantRepo{})

	ctx := appcontext.WithWorkspaceID(context.Background(), wsB)
	_, err := uc.Execute(ctx, deleteReq())
	if err == nil {
		t.Fatalf("dependents in the caller's own workspace must block")
	}
	if !strings.Contains(err.Error(), "1 member") {
		t.Errorf("message should enumerate 1 member, got: %q", err.Error())
	}
	if sg.deleteCalls != 0 {
		t.Errorf("delete must NOT be called when blocked, got %d calls", sg.deleteCalls)
	}
}
