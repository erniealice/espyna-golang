package work_request

import (
	"context"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// GetRequestInboxKPIsRepositories groups all repository dependencies.
type GetRequestInboxKPIsRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// GetRequestInboxKPIsServices groups all business service dependencies.
type GetRequestInboxKPIsServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// RequestInboxKPIs holds the computed KPI projections for the inbox summary row.
type RequestInboxKPIs struct {
	TotalOpen    int
	AssignedToMe int
	Breached     int
	HighPriority int
}

// GetRequestInboxKPIsRequest is the Go-shaped input.
type GetRequestInboxKPIsRequest struct{}

// GetRequestInboxKPIsUseCase computes inbox KPI projections:
//   - TotalOpen: count of non-terminal requests
//   - AssignedToMe: count assigned to the current workspace_user
//   - Breached: count where sla_due_at < now AND status NOT IN (5,6,7) — the
//     SAME single predicate used by the stamp/sweep and partial index
//   - HighPriority: count where priority = 1 AND non-terminal
//
// This is a pure projection (NOT a grouping-proto dashboard — STR-2).
// The adapter MAY implement this as COUNT queries for efficiency.
type GetRequestInboxKPIsUseCase struct {
	repositories GetRequestInboxKPIsRepositories
	services     GetRequestInboxKPIsServices
}

func NewGetRequestInboxKPIsUseCase(repositories GetRequestInboxKPIsRepositories, services GetRequestInboxKPIsServices) *GetRequestInboxKPIsUseCase {
	return &GetRequestInboxKPIsUseCase{repositories: repositories, services: services}
}

func (uc *GetRequestInboxKPIsUseCase) Execute(ctx context.Context, req *GetRequestInboxKPIsRequest) (*RequestInboxKPIs, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// List all requests for the workspace.
	listResp, err := uc.repositories.WorkRequest.ListWorkRequests(ctx, &work_requestpb.ListWorkRequestsRequest{})
	if err != nil {
		return nil, err
	}
	if listResp == nil {
		return &RequestInboxKPIs{}, nil
	}

	now := time.Now().UnixMilli()

	// Resolve the current workspace_user_id from context for "assigned to me".
	currentWorkspaceUserID := contextutil.ExtractWorkspaceUserIDFromContext(ctx)

	kpis := &RequestInboxKPIs{}
	for _, wr := range listResp.Data {
		if isTerminalStatus(wr.Status) {
			continue
		}

		kpis.TotalOpen++

		// Assigned to me: compare assigned_to_workspace_user_id to session.
		if wr.AssignedToWorkspaceUserId != nil && *wr.AssignedToWorkspaceUserId == currentWorkspaceUserID {
			kpis.AssignedToMe++
		}

		// Breached: sla_due_at < now AND non-terminal (same predicate as stamp).
		if wr.SlaDueAt != nil && *wr.SlaDueAt < now {
			kpis.Breached++
		}

		// High priority.
		if wr.Priority == 1 {
			kpis.HighPriority++
		}
	}

	return kpis, nil
}
