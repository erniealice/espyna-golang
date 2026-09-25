package outcome_matrix

import (
	"context"
	"errors"
	"testing"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// ResolveCellRatingDescriptionsUseCase gates NO permission of its own
// (interfaces.md §5 — it runs inside the record action's already-authorized
// task_outcome create/update MINE scope), so these tests cover its actual
// contract: pure pass-through status mapping when a port is wired, and a
// fail-closed PER-CELL result (never an empty response) when it is not — the
// "tenant denial" case: with no provider registered, nothing about the
// caller's tenant/session can be proven, so every requested cell must come
// back UNRESOLVED_IDENTITY rather than silently vanishing (RD-66).

// resolveCellFakePort implements the generated OutcomeMatrixServiceServer via
// the mandatory Unimplemented embed and overrides only the method under test
// (same shape as gateRollupFakePort in get_phase_approval_gate_rollup_test.go).
type resolveCellFakePort struct {
	matrixpb.UnimplementedOutcomeMatrixServiceServer
	called bool
	gotReq *matrixpb.ResolveCellRatingDescriptionsRequest
	resp   *matrixpb.ResolveCellRatingDescriptionsResponse
	err    error
}

func (p *resolveCellFakePort) ResolveCellRatingDescriptions(
	_ context.Context,
	req *matrixpb.ResolveCellRatingDescriptionsRequest,
) (*matrixpb.ResolveCellRatingDescriptionsResponse, error) {
	p.called = true
	p.gotReq = req
	if p.err != nil {
		return nil, p.err
	}
	if p.resp != nil {
		return p.resp, nil
	}
	return &matrixpb.ResolveCellRatingDescriptionsResponse{Success: true}, nil
}

func newResolveCellUC(port query) *ResolveCellRatingDescriptionsUseCase {
	return NewUseCases(Repositories{Query: port}, Services{}).ResolveCellRatingDescriptions
}

func twoCellRequest() *matrixpb.ResolveCellRatingDescriptionsRequest {
	return &matrixpb.ResolveCellRatingDescriptionsRequest{
		Cells: []*matrixpb.CellRatingRef{
			{JobId: "job-1", JobTaskId: "task-1", OutcomeCriteriaId: "crit-A"},
			{JobId: "job-1", JobTaskId: "task-2", OutcomeCriteriaId: "crit-B"},
		},
	}
}

// TestResolveCellRatingDescriptions_NilOrEmptyRequest_EmptySuccessNoPortCall:
// a nil request, or one with zero cells, is a no-op success that never
// touches the port (nothing was asked, so nothing needs resolving).
func TestResolveCellRatingDescriptions_NilOrEmptyRequest_EmptySuccessNoPortCall(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  *matrixpb.ResolveCellRatingDescriptionsRequest
	}{
		{"nil request", nil},
		{"zero cells", &matrixpb.ResolveCellRatingDescriptionsRequest{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port := &resolveCellFakePort{}
			uc := newResolveCellUC(port)
			resp, err := uc.Execute(context.Background(), tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !resp.GetSuccess() || len(resp.GetResults()) != 0 {
				t.Errorf("want empty success, got %+v", resp)
			}
			if port.called {
				t.Error("an empty request must never reach the port")
			}
		})
	}
}

// TestResolveCellRatingDescriptions_DelegatesToPort_StatusMapping proves the
// use case is a pure pass-through: whatever per-cell statuses the adapter
// resolved come back on the response UNCHANGED (the use case adds no
// permission gate and does no re-interpretation of the adapter's decision).
func TestResolveCellRatingDescriptions_DelegatesToPort_StatusMapping(t *testing.T) {
	req := twoCellRequest()
	want := &matrixpb.ResolveCellRatingDescriptionsResponse{
		Success: true,
		Results: []*matrixpb.CellRatingResolution{
			{
				Cell:                   req.Cells[0],
				Status:                 enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED,
				RatingDescriptionSetId: "set-1",
				Descriptions: []*matrixpb.RatingDescription{
					{Description: "Emerging"},
				},
			},
			{
				Cell:   req.Cells[1],
				Status: enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_NO_LINK,
				Reason: "no active rating-description link for this offering and academic year",
			},
		},
	}
	port := &resolveCellFakePort{resp: want}
	uc := newResolveCellUC(port)

	resp, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !port.called || port.gotReq != req {
		t.Fatalf("port must receive the caller's exact request, called=%v", port.called)
	}
	if resp != want {
		t.Fatalf("Execute must return the port's response unchanged, got %+v", resp)
	}
	if len(resp.GetResults()) != 2 {
		t.Fatalf("expected exactly one result per requested cell (RD-66), got %d", len(resp.GetResults()))
	}
	if resp.GetResults()[0].GetStatus() != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED {
		t.Errorf("cell 0 status = %v, want RESOLVED", resp.GetResults()[0].GetStatus())
	}
	if resp.GetResults()[1].GetStatus() != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_NO_LINK {
		t.Errorf("cell 1 status = %v, want NO_LINK", resp.GetResults()[1].GetStatus())
	}
}

// TestResolveCellRatingDescriptions_PortErrorPropagates: a provider failure
// is never swallowed into a success the caller would misread as "resolved".
func TestResolveCellRatingDescriptions_PortErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	port := &resolveCellFakePort{err: sentinel}
	uc := newResolveCellUC(port)

	if _, err := uc.Execute(context.Background(), twoCellRequest()); !errors.Is(err, sentinel) {
		t.Fatalf("port error must propagate, got %v", err)
	}
}

// TestResolveCellRatingDescriptions_NilPort_FailClosedPerCell is the "tenant
// denial" case: with no provider registered nothing about the caller's
// workspace can be proven, so every requested cell must come back
// UNRESOLVED_IDENTITY — never an empty Results slice (which a caller could
// misread as "resolved, nothing to say") and never fewer results than
// requested cells (RD-66).
func TestResolveCellRatingDescriptions_NilPort_FailClosedPerCell(t *testing.T) {
	uc := newResolveCellUC(nil)
	req := twoCellRequest()

	resp, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("nil port must still return Success (fail-closed per cell, not a hard error): %+v", resp)
	}
	if len(resp.GetResults()) != len(req.GetCells()) {
		t.Fatalf("want exactly one result per requested cell, got %d for %d cells", len(resp.GetResults()), len(req.GetCells()))
	}
	for i, r := range resp.GetResults() {
		if r.GetCell() != req.Cells[i] {
			t.Errorf("result %d must echo the requested cell", i)
		}
		if r.GetStatus() != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY {
			t.Errorf("result %d status = %v, want UNRESOLVED_IDENTITY (fail closed)", i, r.GetStatus())
		}
		if r.GetReason() == "" {
			t.Errorf("result %d must carry a non-empty reason", i)
		}
	}
}
