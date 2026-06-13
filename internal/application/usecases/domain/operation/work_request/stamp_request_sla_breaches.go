package work_request

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// StampRequestSLABreachesRepositories groups all repository dependencies.
type StampRequestSLABreachesRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// StampRequestSLABreachesServices groups all business service dependencies.
type StampRequestSLABreachesServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// StampRequestSLABreachesUseCase sweeps open requests past sla_due_at and
// idempotently stamps sla_breached_at.
//
// The sweep uses the SINGLE lifecycle predicate everywhere (RT-3):
//
//	WHERE sla_due_at < now
//	  AND status NOT IN (DECLINED=5, COMPLETED=6, CANCELLED=7)
//	  AND sla_breached_at IS NULL
//
// This is the SAME predicate used by the active flag, the breach KPI
// projection, and the partial SLA index. No drift between surfaces.
//
// The stamp is idempotent: sla_breached_at is set to sla_due_at on the first
// observation and never changed thereafter.
type StampRequestSLABreachesUseCase struct {
	repositories StampRequestSLABreachesRepositories
	services     StampRequestSLABreachesServices
}

func NewStampRequestSLABreachesUseCase(repositories StampRequestSLABreachesRepositories, services StampRequestSLABreachesServices) *StampRequestSLABreachesUseCase {
	return &StampRequestSLABreachesUseCase{repositories: repositories, services: services}
}

// StampRequestSLABreachesRequest is the Go-shaped input. The sweep is
// workspace-scoped (workspace_id from context).
type StampRequestSLABreachesRequest struct{}

// StampRequestSLABreachesResponse reports how many rows were stamped.
type StampRequestSLABreachesResponse struct {
	StampedCount int
}

func (uc *StampRequestSLABreachesUseCase) Execute(ctx context.Context, req *StampRequestSLABreachesRequest) (*StampRequestSLABreachesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// List open (non-terminal) requests that have an SLA due date.
	listResp, err := uc.repositories.WorkRequest.ListWorkRequests(ctx, &work_requestpb.ListWorkRequestsRequest{})
	if err != nil {
		return nil, err
	}
	if listResp == nil {
		return &StampRequestSLABreachesResponse{StampedCount: 0}, nil
	}

	now := time.Now().UnixMilli()
	var toStamp []*work_requestpb.WorkRequest

	for _, wr := range listResp.Data {
		// Single predicate: status NOT IN (5,6,7) AND sla_due_at < now AND sla_breached_at IS NULL.
		if isTerminalStatus(wr.Status) {
			continue
		}
		if wr.SlaDueAt == nil || *wr.SlaDueAt >= now {
			continue
		}
		if wr.SlaBreachedAt != nil {
			continue // already stamped (idempotent)
		}
		toStamp = append(toStamp, wr)
	}

	if len(toStamp) == 0 {
		return &StampRequestSLABreachesResponse{StampedCount: 0}, nil
	}

	// Stamp sla_breached_at = sla_due_at (the moment the SLA was due, not now).
	persist := func(c context.Context) error {
		for _, wr := range toStamp {
			breachedAt := *wr.SlaDueAt
			wr.SlaBreachedAt = &breachedAt
			nowMilli := time.Now().UnixMilli()
			nowStr := time.Now().Format(time.RFC3339)
			wr.DateModified = &nowMilli
			wr.DateModifiedString = &nowStr
			if _, updateErr := uc.repositories.WorkRequest.UpdateWorkRequest(c, &work_requestpb.UpdateWorkRequestRequest{Data: wr}); updateErr != nil {
				return updateErr
			}
		}
		return nil
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, persist); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.sla_stamp_failed", "SLA breach stamp failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := persist(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.sla_stamp_failed", "SLA breach stamp failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	}

	return &StampRequestSLABreachesResponse{StampedCount: len(toStamp)}, nil
}

// SLABreachSweepSQL is a reference for the adapter-level bulk UPDATE that is
// more efficient than the use-case-level loop above. The adapter MAY implement
// this as a single UPDATE statement:
//
//	UPDATE work_request
//	SET sla_breached_at = sla_due_at,
//	    date_modified = NOW()
//	WHERE workspace_id = $1
//	  AND sla_due_at < $2
//	  AND status NOT IN (5,6,7)
//	  AND sla_breached_at IS NULL
//
// The partial index idx_work_request_open_sla covers this predicate exactly.
