package job_phase

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// Plan 20260924-approval-role-workflow D3/D4: the use cases resolve the approval
// scope + policy capabilities fresh inside the transaction and thread them to
// the adapter; the submit override no longer follows publish for STAFF sessions.

// scopeRepo records the ScopeDecision each verify/publish/return observed.
type scopeRepo struct {
	capturingRepo
	verifyDec, publishDec, returnDec approvalctx.ScopeDecision
	verifyOK, publishOK, returnOK    bool
}

func (r *scopeRepo) VerifyJobPhaseApproval(ctx context.Context, req *pb.VerifyJobPhaseApprovalRequest) (*pb.VerifyJobPhaseApprovalResponse, error) {
	r.verifyDec, r.verifyOK = approvalctx.ScopeDecisionFromContext(ctx)
	return r.capturingRepo.VerifyJobPhaseApproval(ctx, req)
}
func (r *scopeRepo) PublishJobPhaseApproval(ctx context.Context, _ *pb.PublishJobPhaseApprovalRequest) (*pb.PublishJobPhaseApprovalResponse, error) {
	r.publishDec, r.publishOK = approvalctx.ScopeDecisionFromContext(ctx)
	return &pb.PublishJobPhaseApprovalResponse{Success: true, Status: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED}, nil
}
func (r *scopeRepo) ReturnJobPhaseApproval(ctx context.Context, req *pb.ReturnJobPhaseApprovalRequest) (*pb.ReturnJobPhaseApprovalResponse, error) {
	r.returnDec, r.returnOK = approvalctx.ScopeDecisionFromContext(ctx)
	return r.capturingRepo.ReturnJobPhaseApproval(ctx, req)
}

func ctxWithKind(uid, ws string, kind int32, principalID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: uid, WorkspaceID: ws, PrincipalType: kind, PrincipalID: principalID,
	})
}

func ctxWithOperator(uid, ws string) context.Context {
	return ctxWithKind(uid, ws, principalTypeOperatorStaff, "wu-1")
}

func ctxWithStaff(uid, ws, staffID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		UserID: uid, WorkspaceID: ws, PrincipalType: principalTypeStaff, PrincipalID: staffID,
	})
}

func TestSubmit_AdminOverride_ByPrincipalKind(t *testing.T) {
	both := map[string]bool{"job_phase:publish": true, permApprovalScopeWorkspace: true}
	cases := []struct {
		name  string
		kind  int32
		perms map[string]bool
		want  bool
	}{
		{"staff + publish only -> no override (publish no longer implies submit-anything)", principalTypeStaff,
			map[string]bool{"job_phase:publish": true}, false},
		{"staff + approval_scope:workspace -> override", principalTypeStaff,
			map[string]bool{permApprovalScopeWorkspace: true}, true},
		{"staff + nothing -> no override", principalTypeStaff, map[string]bool{}, false},
		{"operator staff + publish -> override (preserved D4)", principalTypeOperatorStaff,
			map[string]bool{"job_phase:publish": true}, true},
		{"operator owner + publish -> override (preserved D4)", principalTypeOperatorOwner,
			map[string]bool{"job_phase:publish": true}, true},
		{"operator + approval_scope:workspace -> override", principalTypeOperatorStaff,
			map[string]bool{permApprovalScopeWorkspace: true}, true},
		{"operator + nothing -> no override", principalTypeOperatorStaff, map[string]bool{}, false},
		{"unresolved kind 0 + both -> no override (fail closed)", 0, both, false},
		{"client kind 3 + both -> no override", 3, both, false},
		{"client delegate kind 4 + both -> no override", 4, both, false},
		{"supplier kind 5 + both -> no override", 5, both, false},
		{"supplier delegate kind 6 + both -> no override", 6, both, false},
	}
	// Malformed STAFF (kind 7, empty principal id) never gets the override.
	t.Run("malformed staff (empty principal id) + both -> no override", func(t *testing.T) {
		perms := map[string]bool{"job_phase:submit": true}
		for k, v := range both {
			perms[k] = v
		}
		repo := &capturingRepo{}
		uc := newUC(t, &fakeAuthorizer{enabled: true, strict: true, perms: perms}, &fakeTransactor{supports: true}, repo)
		if _, err := uc.SubmitJobPhaseApproval.Execute(ctxWithKind("u1", "ws1", principalTypeStaff, ""),
			&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.submitDecision.AdminOverride {
			t.Fatal("malformed staff session must not get the submit override")
		}
	})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			perms := map[string]bool{"job_phase:submit": true}
			for k, v := range tc.perms {
				perms[k] = v
			}
			repo := &capturingRepo{}
			uc := newUC(t, &fakeAuthorizer{enabled: true, strict: true, perms: perms}, &fakeTransactor{supports: true}, repo)
			ctx := ctxWithKind("u1", "ws1", tc.kind, "p-1")
			if _, err := uc.SubmitJobPhaseApproval.Execute(ctx,
				&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !repo.submitDecOK || repo.submitDecision.AdminOverride != tc.want {
				t.Fatalf("AdminOverride = %v (ok=%v), want %v", repo.submitDecision.AdminOverride, repo.submitDecOK, tc.want)
			}
		})
	}
}

func TestTransitions_ScopeDecisionThreaded(t *testing.T) {
	all := map[string]bool{
		"job_phase:verify": true, "job_phase:publish": true, "job_phase:return": true,
		permApprovalScopeWorkspace: true, permVerifyOwn: true, permPublishUnverified: true,
	}
	repo := &scopeRepo{}
	uc := newUC(t, &fakeAuthorizer{enabled: true, strict: true, perms: all}, &fakeTransactor{supports: true}, repo)
	ctx := ctxWithStaff("u1", "ws1", "staff-1")
	if _, err := uc.VerifyJobPhaseApproval.Execute(ctx, &pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := uc.PublishJobPhaseApproval.Execute(ctx, &pb.PublishJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := uc.ReturnJobPhaseApproval.Execute(ctx, &pb.ReturnJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("return: %v", err)
	}
	// Each verb resolves only the capabilities it consults.
	if !repo.verifyOK || repo.verifyDec != (approvalctx.ScopeDecision{WorkspaceScope: true, VerifyOwn: true}) {
		t.Fatalf("verify decision = %+v (ok=%v)", repo.verifyDec, repo.verifyOK)
	}
	if !repo.publishOK || repo.publishDec != (approvalctx.ScopeDecision{WorkspaceScope: true, PublishUnverified: true}) {
		t.Fatalf("publish decision = %+v (ok=%v)", repo.publishDec, repo.publishOK)
	}
	if !repo.returnOK || repo.returnDec != (approvalctx.ScopeDecision{WorkspaceScope: true}) {
		t.Fatalf("return decision = %+v (ok=%v)", repo.returnDec, repo.returnOK)
	}
}

func TestTransitions_ScopeDecisionFailsClosed(t *testing.T) {
	// Verb only, no capabilities: every decision field is false.
	repo := &scopeRepo{}
	uc := newUC(t, &fakeAuthorizer{enabled: true, strict: true, perms: map[string]bool{"job_phase:verify": true}},
		&fakeTransactor{supports: true}, repo)
	if _, err := uc.VerifyJobPhaseApproval.Execute(ctxWithStaff("u1", "ws1", "staff-1"),
		&pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !repo.verifyOK || repo.verifyDec != (approvalctx.ScopeDecision{}) {
		t.Fatalf("verify decision = %+v, want zero", repo.verifyDec)
	}
}

func TestTransitions_ScopeUsesFreshVerdict(t *testing.T) {
	// Cached perms still show the capability; the fresh in-tx read shows it revoked.
	repo := &scopeRepo{}
	authz := &fakeAuthorizer{enabled: true, strict: true,
		perms:      map[string]bool{"job_phase:return": true, permApprovalScopeWorkspace: true},
		freshPerms: map[string]bool{"job_phase:return": true},
	}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	if _, err := uc.ReturnJobPhaseApproval.Execute(ctxWithStaff("u1", "ws1", "staff-1"),
		&pb.ReturnJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("return: %v", err)
	}
	if repo.returnDec.WorkspaceScope {
		t.Fatal("workspace scope minted from a stale cached verdict; want the fresh revocation honoured")
	}
}

func TestTransitions_MalformedStaffGetsNoScopeDecision(t *testing.T) {
	all := map[string]bool{"job_phase:verify": true, permApprovalScopeWorkspace: true, permVerifyOwn: true}
	repo := &scopeRepo{}
	uc := newUC(t, &fakeAuthorizer{enabled: true, strict: true, perms: all}, &fakeTransactor{supports: true}, repo)
	if _, err := uc.VerifyJobPhaseApproval.Execute(ctxWithKind("u1", "ws1", principalTypeStaff, ""),
		&pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if repo.verifyDec != (approvalctx.ScopeDecision{}) {
		t.Fatalf("malformed staff got a scope decision %+v; want zero", repo.verifyDec)
	}
}
