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
)

// UpdateWorkRequestStatusRequest is the Go-shaped input for status transitions.
type UpdateWorkRequestStatusRequest struct {
	WorkRequestID  string
	NewStatus      work_requestpb.WorkRequestStatus
	ResolutionNote string // optional staff note on approve/decline/complete/return/hold
}

// UpdateWorkRequestStatusRepositories groups all repository dependencies.
type UpdateWorkRequestStatusRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// UpdateWorkRequestStatusServices groups all business service dependencies.
type UpdateWorkRequestStatusServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// UpdateWorkRequestStatusUseCase enforces the status transition matrix (entities.md section 4.2).
//
// Terminal states: DECLINED=5, COMPLETED=6, CANCELLED=7.
// The single active/terminal predicate: active = status NOT IN (5,6,7).
// APPROVED (4) and negative states (8-11) are NON-terminal (active=true).
//
// date_resolved is stamped ONLY on a TERMINAL transition.
//
// NOTE: The APPROVED->COMPLETED resolution that carries an SR-2 seat replacement
// lives in ResolveWorkRequest (service layer), NOT here.
type UpdateWorkRequestStatusUseCase struct {
	repositories UpdateWorkRequestStatusRepositories
	services     UpdateWorkRequestStatusServices
}

func NewUpdateWorkRequestStatusUseCase(repositories UpdateWorkRequestStatusRepositories, services UpdateWorkRequestStatusServices) *UpdateWorkRequestStatusUseCase {
	return &UpdateWorkRequestStatusUseCase{repositories: repositories, services: services}
}

func (uc *UpdateWorkRequestStatusUseCase) Execute(ctx context.Context, req *UpdateWorkRequestStatusRequest) (*work_requestpb.UpdateWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.WorkRequestID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.id_required", "Work request ID is required [DEFAULT]"))
	}
	if req.NewStatus == work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_UNSPECIFIED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.status_required", "New status is required [DEFAULT]"))
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

	// Validate the transition against the matrix.
	if !isValidTransition(wr.Status, req.NewStatus) {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.invalid_transition", "Invalid status transition [DEFAULT]"))
	}

	now := time.Now()

	// Apply the transition.
	wr.Status = req.NewStatus

	// Derive active from the single predicate: active = status NOT IN (5,6,7).
	wr.Active = !isTerminalStatus(req.NewStatus)

	// Resolution note (optional).
	if req.ResolutionNote != "" {
		wr.ResolutionNote = &req.ResolutionNote
	}

	// date_resolved is stamped ONLY on a TERMINAL transition.
	if isTerminalStatus(req.NewStatus) {
		dateResolved := now.UnixMilli()
		wr.DateResolved = &dateResolved
	}

	wr.DateModified = &[]int64{now.UnixMilli()}[0]
	wr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Persist.
	persist := func(c context.Context) error {
		_, updateErr := uc.repositories.WorkRequest.UpdateWorkRequest(c, &work_requestpb.UpdateWorkRequestRequest{Data: wr})
		return updateErr
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, persist); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.status_update_failed", "Work request status update failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := persist(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.status_update_failed", "Work request status update failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	}

	return &work_requestpb.UpdateWorkRequestResponse{Data: []*work_requestpb.WorkRequest{wr}, Success: true}, nil
}

// isTerminalStatus returns true for the terminal set: {DECLINED=5, COMPLETED=6, CANCELLED=7}.
// Everything else — including APPROVED=4 and the 4 negative states 8-11 — is NON-terminal.
func isTerminalStatus(s work_requestpb.WorkRequestStatus) bool {
	switch s {
	case work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_COMPLETED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

// isValidTransition validates the status transition against the matrix
// defined in entities.md section 4.2.
//
// Transition matrix:
//
//	NEW               -> SUBMITTED, CANCELLED
//	SUBMITTED         -> IN_REVIEW, DECLINED, CANCELLED
//	IN_REVIEW         -> RETURNED_FOR_INFO, ON_HOLD, ESCALATED, PENDING_OVERRIDE, APPROVED, DECLINED
//	RETURNED_FOR_INFO -> SUBMITTED, IN_REVIEW, CANCELLED
//	ON_HOLD           -> IN_REVIEW, CANCELLED
//	ESCALATED         -> IN_REVIEW, APPROVED, DECLINED
//	PENDING_OVERRIDE  -> APPROVED, DECLINED, IN_REVIEW
//	APPROVED          -> COMPLETED
//	DECLINED/COMPLETED/CANCELLED -> (terminal, no outbound)
func isValidTransition(from, to work_requestpb.WorkRequestStatus) bool {
	allowed, ok := transitionMatrix[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// transitionMatrix encodes the full transition matrix from entities.md section 4.2.
var transitionMatrix = map[work_requestpb.WorkRequestStatus][]work_requestpb.WorkRequestStatus{
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_NEW: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_SUBMITTED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_SUBMITTED: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_RETURNED_FOR_INFO,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_ON_HOLD,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_ESCALATED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_PENDING_OVERRIDE,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_APPROVED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_RETURNED_FOR_INFO: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_SUBMITTED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_ON_HOLD: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_ESCALATED: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_APPROVED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_PENDING_OVERRIDE: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_APPROVED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED,
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_IN_REVIEW,
	},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_APPROVED: {
		work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_COMPLETED,
	},
	// Terminal states — no outbound transitions.
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_DECLINED:  {},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_COMPLETED: {},
	work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_CANCELLED: {},
}
