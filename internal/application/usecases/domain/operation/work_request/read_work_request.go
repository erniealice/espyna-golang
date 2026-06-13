package work_request

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
)

// ReadWorkRequestRepositories groups all repository dependencies.
type ReadWorkRequestRepositories struct {
	WorkRequest work_requestpb.WorkRequestDomainServiceServer
}

// ReadWorkRequestServices groups all business service dependencies.
type ReadWorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
}

// ReadWorkRequestUseCase reads a single work request by ID.
//
// IDOR scoping: the adapter's query predicate enforces workspace_id scoping.
// For client paths, the origin-aware predicate (client_id = acting_as_client_id
// AND origin = CLIENT_ORIGINATED) is enforced in the adapter/page-data path.
type ReadWorkRequestUseCase struct {
	repositories ReadWorkRequestRepositories
	services     ReadWorkRequestServices
}

func NewReadWorkRequestUseCase(repositories ReadWorkRequestRepositories, services ReadWorkRequestServices) *ReadWorkRequestUseCase {
	return &ReadWorkRequestUseCase{repositories: repositories, services: services}
}

func (uc *ReadWorkRequestUseCase) Execute(ctx context.Context, req *work_requestpb.ReadWorkRequestRequest) (*work_requestpb.ReadWorkRequestResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequest,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workRequest.validation.id_required", "Work request ID is required [DEFAULT]"))
	}
	return uc.repositories.WorkRequest.ReadWorkRequest(ctx, req)
}
