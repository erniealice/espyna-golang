package job_phase

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/approvalctx"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// ---- fakes ----

// fakeAuthorizer implements ports.Authorizer (superset of actiongate.Authorizer)
// AND the transitions' optional strictAuthorizer capability (StrictEnforcement +
// HasPermissionStrict). `strict` models AUTHZ_ENFORCE; `perms` is the effective
// code set used for both the shadow HasPermission and the strict verdict.
type fakeAuthorizer struct {
	enabled bool
	strict  bool
	perms   map[string]bool
	// freshPerms, when non-nil, is the effective code set seen ONLY by the
	// cache-bypassing HasPermissionStrictFresh path — modelling a permission
	// revoked mid-request (the cached `perms` still shows allow, the fresh in-tx
	// read shows the revocation). nil ⇒ fresh == cached (codex P3 §A1 test seam).
	freshPerms map[string]bool
}

func (f *fakeAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.perms[permission], nil
}
func (f *fakeAuthorizer) StrictEnforcement() bool { return f.strict }
func (f *fakeAuthorizer) HasPermissionStrict(_ context.Context, _ string, permission string) (bool, error) {
	return f.perms[permission], nil
}
func (f *fakeAuthorizer) HasPermissionStrictFresh(_ context.Context, _ string, permission string) (bool, error) {
	if f.freshPerms != nil {
		return f.freshPerms[permission], nil
	}
	return f.perms[permission], nil
}
func (f *fakeAuthorizer) HasGlobalPermission(_ context.Context, _ string, permission string) (bool, error) {
	return f.perms[permission], nil
}
func (f *fakeAuthorizer) HasPermissionInWorkspace(_ context.Context, _, _ string, permission string) (bool, error) {
	return f.perms[permission], nil
}
func (f *fakeAuthorizer) GetUserRoles(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (f *fakeAuthorizer) GetUserRolesInWorkspace(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}
func (f *fakeAuthorizer) GetUserWorkspaces(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (f *fakeAuthorizer) GetUserPermissionCodes(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (f *fakeAuthorizer) IsEnabled() bool { return f.enabled }

// fakeTransactor runs the operation directly (single connection) and reports
// whether transactions are supported.
type fakeTransactor struct{ supports bool }

func (f *fakeTransactor) ExecuteInTransaction(ctx context.Context, op func(ctx context.Context) error) error {
	return op(ctx)
}
func (f *fakeTransactor) SupportsTransactions() bool                 { return f.supports }
func (f *fakeTransactor) IsTransactionActive(_ context.Context) bool { return f.supports }

// capturingRepo records the last create/update payload and the submit decision
// it observed in the transition ctx.
type capturingRepo struct {
	pb.UnimplementedJobPhaseDomainServiceServer
	lastCreate     *pb.JobPhase
	lastUpdate     *pb.JobPhase
	submitDecision approvalctx.SubmitDecision
	submitDecOK    bool
	lastReturnReq  *pb.ReturnJobPhaseApprovalRequest
}

func (r *capturingRepo) CreateJobPhase(_ context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	r.lastCreate = req.Data
	return &pb.CreateJobPhaseResponse{Success: true, Data: []*pb.JobPhase{req.Data}}, nil
}
func (r *capturingRepo) UpdateJobPhase(_ context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	r.lastUpdate = req.Data
	return &pb.UpdateJobPhaseResponse{Success: true, Data: []*pb.JobPhase{req.Data}}, nil
}
func (r *capturingRepo) SubmitJobPhaseApproval(ctx context.Context, _ *pb.SubmitJobPhaseApprovalRequest) (*pb.SubmitJobPhaseApprovalResponse, error) {
	r.submitDecision, r.submitDecOK = approvalctx.SubmitDecisionFromContext(ctx)
	return &pb.SubmitJobPhaseApprovalResponse{Success: true, Status: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_FOR_REVIEW, AffectedCount: 3}, nil
}
func (r *capturingRepo) VerifyJobPhaseApproval(_ context.Context, _ *pb.VerifyJobPhaseApprovalRequest) (*pb.VerifyJobPhaseApprovalResponse, error) {
	return &pb.VerifyJobPhaseApprovalResponse{Success: true, Status: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED, AffectedCount: 3}, nil
}
func (r *capturingRepo) ReturnJobPhaseApproval(_ context.Context, req *pb.ReturnJobPhaseApprovalRequest) (*pb.ReturnJobPhaseApprovalResponse, error) {
	r.lastReturnReq = req
	return &pb.ReturnJobPhaseApprovalResponse{Success: true, Status: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS, AffectedCount: 3}, nil
}

func ctxWithUser(uid, ws string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: uid, WorkspaceID: ws})
}

func newUC(t *testing.T, authz *fakeAuthorizer, tx *fakeTransactor, repo pb.JobPhaseDomainServiceServer) *UseCases {
	t.Helper()
	gate := actiongate.NewActionGatekeeper(authz, nil)
	return NewUseCases(
		JobPhaseRepositories{JobPhase: repo},
		JobPhaseServices{
			Authorizer:       authz,
			Transactor:       tx,
			ActionGatekeeper: gate,
		},
	)
}

// ---- transition gate + tx-required ----

func TestSubmit_GateDenied(t *testing.T) {
	authz := &fakeAuthorizer{enabled: true, perms: map[string]bool{}} // no job_phase:submit
	uc := newUC(t, authz, &fakeTransactor{supports: true}, &capturingRepo{})
	_, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
	if err == nil {
		t.Fatal("expected gate denial, got nil")
	}
}

func TestSubmit_TransactionRequired(t *testing.T) {
	authz := &fakeAuthorizer{enabled: false} // gate passes
	uc := newUC(t, authz, &fakeTransactor{supports: false}, &capturingRepo{})
	_, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
	if err == nil {
		t.Fatal("expected fail-closed on missing transaction support, got nil")
	}
}

func TestSubmit_EmptyIdsRejected(t *testing.T) {
	authz := &fakeAuthorizer{enabled: false}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, &capturingRepo{})
	if _, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "", JobTemplatePhaseId: "p1"}); err == nil {
		t.Fatal("expected rejection of empty template id")
	}
}

// ---- A1: fresh, cache-bypassing, in-transaction verb + override ----

// TestSubmit_RevokedInTx_Denied proves the AUTHORITATIVE verb verdict is the
// FRESH in-transaction read, not the warm cache: the pre-tx cached verdict allows
// job_phase:submit, but a permission revoked mid-request (freshPerms) denies it,
// and the transition is refused (codex P3 §A1). The repo transition must NOT run.
func TestSubmit_RevokedInTx_Denied(t *testing.T) {
	authz := &fakeAuthorizer{
		enabled: true, strict: true,
		perms:      map[string]bool{"job_phase:submit": true}, // warm cache: allow
		freshPerms: map[string]bool{"job_phase:submit": false}, // revoked in-tx: deny
	}
	repo := &capturingRepo{}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	_, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
	if err == nil {
		t.Fatal("expected in-tx fresh verdict to deny a revoked submit, got nil")
	}
	if repo.submitDecOK {
		t.Fatal("repo transition ran despite the fresh in-tx deny — verb gate is not authoritative")
	}
}

// TestSubmit_AdminOverride_UsesFreshVerdict proves the publish-derived admin
// override reads the FRESH verdict: cached publish=allow, fresh publish=deny ⇒ NO
// override (fall through to the D7 ownership check), never a stale cache grant.
func TestSubmit_AdminOverride_UsesFreshVerdict(t *testing.T) {
	authz := &fakeAuthorizer{
		enabled: true, strict: true,
		perms:      map[string]bool{"job_phase:submit": true, "job_phase:publish": true},  // warm: publish allow
		freshPerms: map[string]bool{"job_phase:submit": true, "job_phase:publish": false}, // in-tx: publish revoked
	}
	repo := &capturingRepo{}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	_, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.submitDecOK {
		t.Fatal("adapter did not observe a submit decision in ctx")
	}
	if repo.submitDecision.AdminOverride {
		t.Fatal("AdminOverride granted from a stale cached publish verdict — must use the fresh in-tx verdict")
	}
}

// ---- admin-override resolution + threading ----

func TestSubmit_AdminOverrideThreaded(t *testing.T) {
	cases := []struct {
		name        string
		publishPerm bool
		want        bool
	}{
		{"has publish authority -> override", true, true},
		{"no publish authority -> ownership", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// strict=true (AUTHZ_ENFORCE active). The override is the STRICT publish
			// verdict resolved INSIDE the transaction.
			authz := &fakeAuthorizer{enabled: true, strict: true, perms: map[string]bool{
				"job_phase:submit":  true,
				"job_phase:publish": tc.publishPerm,
			}}
			repo := &capturingRepo{}
			uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
			_, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
				&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !repo.submitDecOK {
				t.Fatal("adapter did not observe a submit decision in ctx")
			}
			if repo.submitDecision.AdminOverride != tc.want {
				t.Fatalf("AdminOverride = %v, want %v", repo.submitDecision.AdminOverride, tc.want)
			}
		})
	}
}

// TestSubmit_NonStrictAuthzRefused proves the transition FAILS CLOSED when strict
// enforcement is not active (the shadow-mode allow-on-deny posture is unsafe for
// these verbs). Replaces the prior AuthzDisabledGrantsOverride test whose premise
// (disabled ⇒ grant override) was the CRITICAL fail-open codex flagged.
func TestSubmit_NonStrictAuthzRefused(t *testing.T) {
	// enabled=true so the ActionGatekeeper does not short-circuit; strict=false so
	// requireStrictAuthorizer refuses.
	authz := &fakeAuthorizer{enabled: true, strict: false, perms: map[string]bool{
		"job_phase:submit": true, "job_phase:publish": true,
	}}
	repo := &capturingRepo{}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	if _, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err == nil {
		t.Fatal("expected fail-closed when AUTHZ_ENFORCE is off, got nil")
	}
	if repo.submitDecOK {
		t.Fatal("transition should not have reached the adapter with non-strict authz")
	}
}

// TestSubmit_StrictVerbDenyRefused proves the strict verb re-check denies even
// when the shadow ActionGatekeeper would allow. strict=true but the submit code
// is absent from the effective set → refused (a shadow authorizer would allow).
func TestSubmit_StrictVerbDenyRefused(t *testing.T) {
	authz := &fakeAuthorizer{enabled: true, strict: true, perms: map[string]bool{
		"job_phase:publish": true, // has publish but NOT submit
	}}
	repo := &capturingRepo{}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	if _, err := uc.SubmitJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.SubmitJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err == nil {
		t.Fatal("expected strict-verb denial without job_phase:submit, got nil")
	}
}

// TestTransition_NonStrictAuthorizerTypeRefused proves the production no-op/mock
// authorizer — which does NOT implement the strictAuthorizer capability and whose
// IsEnabled()=false lets the ActionGatekeeper short-circuit — is still refused by
// requireStrictAuthorizer. The transitions never fall back to a shadow/AllowAll
// verdict.
func TestTransition_NonStrictAuthorizerTypeRefused(t *testing.T) {
	authz := ports.NewNoOpAuthorizer() // IsEnabled()==false, no strict methods
	uc := NewUseCases(
		JobPhaseRepositories{JobPhase: &capturingRepo{}},
		JobPhaseServices{
			Authorizer:       authz,
			Transactor:       &fakeTransactor{supports: true},
			ActionGatekeeper: actiongate.NewActionGatekeeper(authz, nil),
		},
	)
	if _, err := uc.VerifyJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err == nil {
		t.Fatal("expected refusal with a non-strict (no-op) authorizer")
	}
}

// ---- verify / return orchestration ----

func TestVerify_HappyPath(t *testing.T) {
	authz := &fakeAuthorizer{enabled: true, strict: true, perms: map[string]bool{"job_phase:verify": true}}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, &capturingRepo{})
	resp, err := uc.VerifyJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetStatus() != pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED || resp.GetAffectedCount() != 3 {
		t.Fatalf("unexpected verify response: %+v", resp)
	}
}

func TestReturn_ReasonPassthrough(t *testing.T) {
	authz := &fakeAuthorizer{enabled: true, strict: true, perms: map[string]bool{"job_phase:return": true}}
	repo := &capturingRepo{}
	uc := newUC(t, authz, &fakeTransactor{supports: true}, repo)
	reason := "grades incomplete"
	if _, err := uc.ReturnJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.ReturnJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1", Reason: &reason}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastReturnReq == nil || repo.lastReturnReq.GetReason() != reason {
		t.Fatalf("reason not passed through: %+v", repo.lastReturnReq)
	}
}

// ---- generic-CRUD forgery hardening (use-case layer) ----

func TestCreateJobPhase_ForgedApprovalFieldsForced(t *testing.T) {
	authz := &fakeAuthorizer{enabled: false}
	repo := &capturingRepo{}
	// nil Transactor -> Create runs the repo directly (no tx branch).
	uc := NewUseCases(
		JobPhaseRepositories{JobPhase: repo},
		JobPhaseServices{Authorizer: authz, ActionGatekeeper: actiongate.NewActionGatekeeper(authz, nil)},
	)
	forgedBy := "attacker"
	forgedAt := int64(123)
	req := &pb.CreateJobPhaseRequest{Data: &pb.JobPhase{
		Name:           "Term 1",
		JobId:          "job-1",
		ApprovalStatus: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED,
		SubmittedBy:    &forgedBy,
		SubmittedAt:    &forgedAt,
		PublishedBy:    &forgedBy,
		PublishedAt:    &forgedAt,
		ReturnReason:   &forgedBy,
	}}
	if _, err := uc.CreateJobPhase.Execute(ctxWithUser("u1", "ws1"), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := repo.lastCreate
	if got.GetApprovalStatus() != pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS {
		t.Fatalf("approval_status = %v, want IN_PROGRESS", got.GetApprovalStatus())
	}
	if got.SubmittedBy != nil || got.SubmittedAt != nil || got.PublishedBy != nil || got.PublishedAt != nil || got.ReturnReason != nil {
		t.Fatalf("forged audit fields were not stripped on create: %+v", got)
	}
}

func TestUpdateJobPhase_ForgedApprovalFieldsStripped(t *testing.T) {
	authz := &fakeAuthorizer{enabled: false}
	repo := &capturingRepo{}
	uc := NewUseCases(
		JobPhaseRepositories{JobPhase: repo},
		JobPhaseServices{Authorizer: authz, ActionGatekeeper: actiongate.NewActionGatekeeper(authz, nil)},
	)
	forgedBy := "attacker"
	forgedAt := int64(123)
	req := &pb.UpdateJobPhaseRequest{Data: &pb.JobPhase{
		Id:             "phase-1",
		Name:           "Term 1",
		ApprovalStatus: pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED,
		VerifiedBy:     &forgedBy,
		VerifiedAt:     &forgedAt,
		ReturnedBy:     &forgedBy,
	}}
	if _, err := uc.UpdateJobPhase.Execute(ctxWithUser("u1", "ws1"), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := repo.lastUpdate
	if got.GetApprovalStatus() != pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED {
		t.Fatalf("approval_status = %v, want UNSPECIFIED (stripped)", got.GetApprovalStatus())
	}
	if got.VerifiedBy != nil || got.VerifiedAt != nil || got.ReturnedBy != nil {
		t.Fatalf("forged audit fields were not stripped on update: %+v", got)
	}
}

// guard: the transition services fail closed when the transactor is entirely absent.
func TestVerify_NilTransactorFailsClosed(t *testing.T) {
	authz := &fakeAuthorizer{enabled: false}
	uc := NewUseCases(
		JobPhaseRepositories{JobPhase: &capturingRepo{}},
		JobPhaseServices{Authorizer: authz, ActionGatekeeper: actiongate.NewActionGatekeeper(authz, nil)},
	)
	if _, err := uc.VerifyJobPhaseApproval.Execute(ctxWithUser("u1", "ws1"),
		&pb.VerifyJobPhaseApprovalRequest{JobTemplateId: "t1", JobTemplatePhaseId: "p1"}); err == nil {
		t.Fatal("expected fail-closed with nil transactor")
	}
}
