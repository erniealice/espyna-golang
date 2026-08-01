package outcome_matrix

import (
	"context"
	"errors"
	"testing"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Gate-rollup use case (plan 20260729 Phase 2): three list gates in a fixed
// order, strict validation, and — THE divergence under test — a nil provider
// port is an ERROR, never the empty success GetOutcomeMatrixUseCase degrades
// to. This response feeds a document-integrity decision; an empty success would
// read as "no sheets".

// recordingAuthorizer is a minimal RBAC port that RECORDS every consulted
// permission code (so gate order is assertable) and answers from a fixed set.
// IsEnabled() is true so Check actually consults HasPermission.
type recordingAuthorizer struct {
	allowed map[string]bool
	asked   []string
}

func (f *recordingAuthorizer) IsEnabled() bool { return true }
func (f *recordingAuthorizer) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	f.asked = append(f.asked, permission)
	return f.allowed[permission], nil
}

// gateRollupFakePort implements the generated OutcomeMatrixServiceServer via
// the mandatory Unimplemented embed and overrides only the method under test.
type gateRollupFakePort struct {
	matrixpb.UnimplementedOutcomeMatrixServiceServer
	called bool
	gotReq *matrixpb.GetPhaseApprovalGateRollupRequest
	resp   *matrixpb.GetPhaseApprovalGateRollupResponse
	err    error
}

func (p *gateRollupFakePort) GetPhaseApprovalGateRollup(
	_ context.Context,
	req *matrixpb.GetPhaseApprovalGateRollupRequest,
) (*matrixpb.GetPhaseApprovalGateRollupResponse, error) {
	p.called = true
	p.gotReq = req
	if p.err != nil {
		return nil, p.err
	}
	if p.resp != nil {
		return p.resp, nil
	}
	return &matrixpb.GetPhaseApprovalGateRollupResponse{Success: true}, nil
}

func gateRollupGate(auth *recordingAuthorizer) *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(auth, nil)
}

func allowAll() *recordingAuthorizer {
	return &recordingAuthorizer{allowed: map[string]bool{
		entityid.EntityPermission(entityid.JobPhase, entityid.ActionList):    true,
		entityid.EntityPermission(entityid.JobTask, entityid.ActionList):     true,
		entityid.EntityPermission(entityid.TaskOutcome, entityid.ActionList): true,
	}}
}

func gateRollupUserCtx() context.Context {
	return contextutil.WithUserID(context.Background(), "user-1")
}

func newGateRollupUC(port query, g *actiongate.ActionGatekeeper) *GetPhaseApprovalGateRollupUseCase {
	return NewUseCases(
		Repositories{Query: port},
		Services{ActionGatekeeper: g},
	).GetPhaseApprovalGateRollup
}

func validGateRollupReq() *matrixpb.GetPhaseApprovalGateRollupRequest {
	return &matrixpb.GetPhaseApprovalGateRollupRequest{
		SubscriptionGroupId: "grp-A",
		JobTemplatePhaseIds: []string{"tp-1", "tp-2"},
	}
}

// TestGateRollup_GateOrderAndDelegation: all three gates granted → the port is
// called with the request unchanged, and the gates were consulted in the fixed
// job_phase → job_task → task_outcome order (the RPC collapses exactly that
// three-entity read walk, so it must demand the same three capabilities).
func TestGateRollup_GateOrderAndDelegation(t *testing.T) {
	auth := allowAll()
	port := &gateRollupFakePort{}
	uc := newGateRollupUC(port, gateRollupGate(auth))

	req := validGateRollupReq()
	resp, err := uc.Execute(gateRollupUserCtx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Error("expected the port's success response")
	}
	if !port.called || port.gotReq != req {
		t.Errorf("port must receive the caller's exact request, called=%v", port.called)
	}
	want := []string{
		entityid.EntityPermission(entityid.JobPhase, entityid.ActionList),
		entityid.EntityPermission(entityid.JobTask, entityid.ActionList),
		entityid.EntityPermission(entityid.TaskOutcome, entityid.ActionList),
	}
	if len(auth.asked) != len(want) {
		t.Fatalf("gates consulted = %v, want %v", auth.asked, want)
	}
	for i := range want {
		if auth.asked[i] != want[i] {
			t.Errorf("gate %d = %q, want %q (fixed order)", i, auth.asked[i], want[i])
		}
	}
}

// TestGateRollup_EachGateDenialIsAnError: denying ANY one of the three list
// capabilities must error and never reach the port — anything less silently
// widens what a partially-permissioned role can learn.
func TestGateRollup_EachGateDenialIsAnError(t *testing.T) {
	for _, deny := range []string{entityid.JobPhase, entityid.JobTask, entityid.TaskOutcome} {
		t.Run("deny "+deny, func(t *testing.T) {
			auth := allowAll()
			auth.allowed[entityid.EntityPermission(deny, entityid.ActionList)] = false
			port := &gateRollupFakePort{}
			uc := newGateRollupUC(port, gateRollupGate(auth))

			if _, err := uc.Execute(gateRollupUserCtx(), validGateRollupReq()); err == nil {
				t.Fatal("denied gate must error")
			}
			if port.called {
				t.Error("a denied gate must never reach the port")
			}
		})
	}
}

// TestGateRollup_NilGatekeeperDenies: Check's nil-receiver guard DENIES, so a
// mis-wired nil gatekeeper is an error, not a skipped gate.
func TestGateRollup_NilGatekeeperDenies(t *testing.T) {
	port := &gateRollupFakePort{}
	uc := newGateRollupUC(port, nil)
	if _, err := uc.Execute(gateRollupUserCtx(), validGateRollupReq()); err == nil {
		t.Fatal("nil gatekeeper must deny")
	}
	if port.called {
		t.Error("nil gatekeeper must never reach the port")
	}
}

// TestGateRollup_ValidationErrors: nil request, empty group id and empty
// template-phase ids each refuse before the port.
func TestGateRollup_ValidationErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  *matrixpb.GetPhaseApprovalGateRollupRequest
	}{
		{"nil request", nil},
		{"empty group id", &matrixpb.GetPhaseApprovalGateRollupRequest{JobTemplatePhaseIds: []string{"tp-1"}}},
		{"empty template phase ids", &matrixpb.GetPhaseApprovalGateRollupRequest{SubscriptionGroupId: "grp-A"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port := &gateRollupFakePort{}
			uc := newGateRollupUC(port, gateRollupGate(allowAll()))
			if _, err := uc.Execute(gateRollupUserCtx(), tc.req); err == nil {
				t.Fatal("invalid request must error")
			}
			if port.called {
				t.Error("invalid request must never reach the port")
			}
		})
	}
}

// TestGateRollup_NilPortIsAnErrorNotEmptySuccess is THE divergence test: with
// no registered provider the use case must ERROR — the deliberate inversion of
// GetOutcomeMatrixUseCase's empty-success degrade. An empty success here would
// read as "no sheets" to the document-integrity consumer.
func TestGateRollup_NilPortIsAnErrorNotEmptySuccess(t *testing.T) {
	uc := newGateRollupUC(nil, gateRollupGate(allowAll()))
	resp, err := uc.Execute(gateRollupUserCtx(), validGateRollupReq())
	if err == nil {
		t.Fatalf("nil port must be an ERROR (fail closed), got resp=%+v", resp)
	}
	if resp != nil {
		t.Errorf("nil port must not return a response, got %+v", resp)
	}
}

// TestGateRollup_PortErrorPropagates: a provider failure is never swallowed
// into a success the consumer would misread.
func TestGateRollup_PortErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	port := &gateRollupFakePort{err: sentinel}
	uc := newGateRollupUC(port, gateRollupGate(allowAll()))
	if _, err := uc.Execute(gateRollupUserCtx(), validGateRollupReq()); !errors.Is(err, sentinel) {
		t.Fatalf("port error must propagate, got %v", err)
	}
}
