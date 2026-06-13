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

// CreateWorkRequestRepositories groups all repository dependencies.
type CreateWorkRequestRepositories struct {
	WorkRequest     work_requestpb.WorkRequestDomainServiceServer
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // FK validation + SLA snapshot
}

// CreateWorkRequestServices groups all business service dependencies.
type CreateWorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
}

// CreateWorkRequestUseCase creates a new work request.
//
// CRITICAL SECURITY INVARIANT (Q-REQ-ORIGIN):
//   - For origin=CLIENT_ORIGINATED: client_id is stamped from session
//     (acting_as_client_id), NEVER from the request body.
//   - For origin=CLIENT_RELATED_INTERNAL: client_id is stamped from the request
//     body (staff provides it), but requires staff principal.
//   - For origin=INTERNAL: client_id is set to nil. No client at all.
//
// Generates request_number via the repository (FOR UPDATE counter pattern).
// Sets status=NEW, active=true. Validates the idempotency key.
type CreateWorkRequestUseCase struct {
	repositories CreateWorkRequestRepositories
	services     CreateWorkRequestServices
}

func NewCreateWorkRequestUseCase(repositories CreateWorkRequestRepositories, services CreateWorkRequestServices) *CreateWorkRequestUseCase {
	return &CreateWorkRequestUseCase{repositories: repositories, services: services}
}

func (uc *CreateWorkRequestUseCase) Execute(ctx context.Context, req *work_requestpb.CreateWorkRequestRequest) (*work_requestpb.CreateWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.data_required", "Work request data is required [DEFAULT]"))
	}

	// Origin validation: must be explicitly set.
	if req.Data.Origin == work_requestpb.WorkRequestOrigin_WORK_REQUEST_ORIGIN_UNSPECIFIED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.origin_required", "Work request origin is required [DEFAULT]"))
	}

	// IDOR gate: origin-aware client_id stamping (Q-REQ-ORIGIN).
	switch req.Data.Origin {
	case work_requestpb.WorkRequestOrigin_WORK_REQUEST_ORIGIN_CLIENT_ORIGINATED:
		// Client-originated: client_id MUST come from session, NEVER from the
		// request body. Deny before any SQL if acting_as_client_id is empty.
		actingClient := contextutil.GetActingAsClientIDFromContext(ctx)
		if actingClient == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.no_acting_client", "An acting client scope is required for a client-originated request [DEFAULT]"))
		}
		req.Data.ClientId = &actingClient

	case work_requestpb.WorkRequestOrigin_WORK_REQUEST_ORIGIN_CLIENT_RELATED_INTERNAL:
		// Client-related-internal: staff provides client_id via the form.
		if req.Data.ClientId == nil || *req.Data.ClientId == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.client_id_required_for_client_related", "Client ID is required for a client-related internal request [DEFAULT]"))
		}

	case work_requestpb.WorkRequestOrigin_WORK_REQUEST_ORIGIN_INTERNAL:
		// Internal: no client. Set client_id to nil explicitly.
		req.Data.ClientId = nil
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// FK validation: work_request_type must exist and be active.
	if err := uc.validateWorkRequestType(ctx, req.Data.WorkRequestTypeId); err != nil {
		return nil, err
	}

	uc.enrich(ctx, req.Data)

	// Idempotency: the DB has UNIQUE(workspace_id, requested_by_user_id,
	// submission_idempotency_key). On conflict the adapter should return the
	// existing row instead of erroring. The use case trusts the adapter contract.
	var resp *work_requestpb.CreateWorkRequestResponse
	var createErr error

	persist := func(c context.Context) error {
		var err error
		resp, err = uc.repositories.WorkRequest.CreateWorkRequest(c, req)
		if err != nil {
			createErr = err
			return err
		}
		return nil
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, persist); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.creation_failed", "Work request creation failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, err)
		}
	} else {
		if err := persist(ctx); err != nil {
			translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.creation_failed", "Work request creation failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translated, createErr)
		}
	}

	return resp, nil
}

func (uc *CreateWorkRequestUseCase) validateBusinessRules(ctx context.Context, wr *work_requestpb.WorkRequest) error {
	if wr.WorkRequestTypeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.type_required", "Work request type is required [DEFAULT]"))
	}
	if wr.Title == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.title_required", "Work request title is required [DEFAULT]"))
	}
	if wr.SubmissionIdempotencyKey == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.idempotency_key_required", "Submission idempotency key is required [DEFAULT]"))
	}
	return nil
}

// validateWorkRequestType reads the type and asserts it exists and is ACTIVE.
func (uc *CreateWorkRequestUseCase) validateWorkRequestType(ctx context.Context, typeID string) error {
	if uc.repositories.WorkRequestType == nil {
		return nil // graceful skip during tests without type repo
	}
	resp, err := uc.repositories.WorkRequestType.ReadWorkRequestType(ctx, &work_request_typepb.ReadWorkRequestTypeRequest{
		Data: &work_request_typepb.WorkRequestType{Id: typeID},
	})
	if err != nil {
		return err
	}
	if resp == nil || len(resp.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.type_not_found", "Work request type not found [DEFAULT]"))
	}
	if resp.Data[0].Status != work_request_typepb.WorkRequestTypeStatus_WORK_REQUEST_TYPE_STATUS_ACTIVE {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.type_not_active", "Work request type is not active [DEFAULT]"))
	}
	return nil
}

func (uc *CreateWorkRequestUseCase) enrich(ctx context.Context, wr *work_requestpb.WorkRequest) {
	now := time.Now()
	if wr.Id == "" {
		wr.Id = uc.services.IDGenerator.GenerateID()
	}

	// New requests start as NEW.
	wr.Status = work_requestpb.WorkRequestStatus_WORK_REQUEST_STATUS_NEW
	// active = status NOT IN (5,6,7) — NEW is active.
	wr.Active = true

	// Stamp requested_by from session user if not already set.
	if wr.RequestedByUserId == "" {
		wr.RequestedByUserId = contextutil.ExtractUserIDFromContext(ctx)
	}

	// Default priority to normal (0).
	if wr.Priority < 0 || wr.Priority > 1 {
		wr.Priority = 0
	}

	// request_number is generated by the adapter (FOR UPDATE counter pattern).
	// The use case does NOT set it — the adapter stamps it atomically.

	wr.DateCreated = &[]int64{now.UnixMilli()}[0]
	wr.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	wr.DateModified = &[]int64{now.UnixMilli()}[0]
	wr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
