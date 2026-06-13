package work_request

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// SubmitWorkRequestRequest is the Go-shaped input (no proto request type exists
// for this state-transition op).
type SubmitWorkRequestRequest struct {
	WorkRequestID string
}

// SubmitWorkRequestRepositories groups all repository dependencies.
type SubmitWorkRequestRepositories struct {
	WorkRequest     work_requestpb.WorkRequestDomainServiceServer
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // SLA snapshot source
}

// SubmitWorkRequestServices groups all business service dependencies.
type SubmitWorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// SubmitWorkRequestUseCase transitions a work request from NEW to SUBMITTED.
//
// On submit:
//   - Validates the current status is NEW (only NEW -> SUBMITTED is allowed).
//   - Snapshots sla_target_hours from the linked WorkRequestType.default_sla_hours.
//   - Stamps sla_due_at = now + sla_target_hours (unix millis).
//   - Stamps date_submitted.
//   - active stays true (SUBMITTED is non-terminal).
type SubmitWorkRequestUseCase struct {
	repositories SubmitWorkRequestRepositories
	services     SubmitWorkRequestServices
}

func NewSubmitWorkRequestUseCase(repositories SubmitWorkRequestRepositories, services SubmitWorkRequestServices) *SubmitWorkRequestUseCase {
	return &SubmitWorkRequestUseCase{repositories: repositories, services: services}
}

func (uc *SubmitWorkRequestUseCase) Execute(ctx context.Context, req *SubmitWorkRequestRequest) (*work_requestpb.UpdateWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.WorkRequestID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.id_required", "Work request ID is required [DEFAULT]"))
	}

	// Load the current work request.
	readResp, err := uc.repositories.WorkRequest.ReadWorkRequest(ctx, &work_requestpb.ReadWorkRequestRequest{
		Data: &work_requestpb.WorkRequest{Id: req.WorkRequestID},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.not_found", "Work request not found [DEFAULT]"))
	}
	wr := readResp.Data[0]

	// Only NEW may be submitted.
	if wr.Status != work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_NEW {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.not_new", "Only a new work request can be submitted [DEFAULT]"))
	}

	now := time.Now()

	// Snapshot SLA from the type.
	slaHours := uc.resolveSLAHours(ctx, wr.WorkRequestTypeId)
	wr.SlaTargetHours = slaHours

	// sla_due_at = now + sla_target_hours (in unix millis).
	if slaHours > 0 {
		dueAt := now.Add(time.Duration(slaHours) * time.Hour).UnixMilli()
		wr.SlaDueAt = &dueAt
	}

	// Transition to SUBMITTED.
	wr.Status = work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_SUBMITTED
	// active = status NOT IN (5,6,7) — SUBMITTED is active.
	wr.Active = true

	// Stamp date_submitted.
	dateSubmitted := now.UnixMilli()
	wr.DateSubmitted = &dateSubmitted
	wr.DateModified = &[]int64{now.UnixMilli()}[0]
	wr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Persist.
	persist := func(c context.Context) error {
		_, updateErr := uc.repositories.WorkRequest.UpdateWorkRequest(c, &work_requestpb.UpdateWorkRequestRequest{Data: wr})
		return updateErr
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, persist); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.submit_failed", "Work request submit failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := persist(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.submit_failed", "Work request submit failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	}

	return &work_requestpb.UpdateWorkRequestResponse{Data: []*work_requestpb.WorkRequest{wr}, Success: true}, nil
}

// resolveSLAHours reads the type's default_sla_hours. Falls back to 0 if the
// type cannot be read (graceful degradation).
func (uc *SubmitWorkRequestUseCase) resolveSLAHours(ctx context.Context, typeID string) int64 {
	if uc.repositories.WorkRequestType == nil || typeID == "" {
		return 0
	}
	resp, err := uc.repositories.WorkRequestType.ReadWorkRequestType(ctx, &work_request_typepb.ReadWorkRequestTypeRequest{
		Data: &work_request_typepb.WorkRequestType{Id: typeID},
	})
	if err != nil || resp == nil || len(resp.Data) == 0 {
		return 0
	}
	return resp.Data[0].DefaultSlaHours
}
