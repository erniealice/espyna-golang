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

// UpdateWorkRequestRepositories groups all repository dependencies.
type UpdateWorkRequestRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// UpdateWorkRequestServices groups all business service dependencies.
type UpdateWorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// UpdateWorkRequestUseCase updates a work request's editable fields.
//
// Used for the Info tab edit and the RETURNED_FOR_INFO resubmit path
// (the transition matrix requires UpdateWorkRequest for field edits;
// UpdateWorkRequestStatus handles status transitions separately).
// This use case does NOT change status — it only updates content fields
// (title, description, payload_json, priority, etc.).
type UpdateWorkRequestUseCase struct {
	repositories UpdateWorkRequestRepositories
	services     UpdateWorkRequestServices
}

func NewUpdateWorkRequestUseCase(repositories UpdateWorkRequestRepositories, services UpdateWorkRequestServices) *UpdateWorkRequestUseCase {
	return &UpdateWorkRequestUseCase{repositories: repositories, services: services}
}

func (uc *UpdateWorkRequestUseCase) Execute(ctx context.Context, req *work_requestpb.UpdateWorkRequestRequest) (*work_requestpb.UpdateWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.id_required", "Work request ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	resp, err := uc.repositories.WorkRequest.UpdateWorkRequest(ctx, req)
	if err != nil {
		translated := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.errors.update_failed", "Work request update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translated, err)
	}
	return resp, nil
}
