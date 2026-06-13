package disbursement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// DeleteDisbursementRepositories groups all repository dependencies
type DeleteDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// DeleteDisbursementServices groups all business service dependencies
type DeleteDisbursementServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteDisbursementUseCase handles the business logic for deleting disbursements
type DeleteDisbursementUseCase struct {
	repositories DeleteDisbursementRepositories
	services     DeleteDisbursementServices
}

// NewDeleteDisbursementUseCase creates a new DeleteDisbursementUseCase
func NewDeleteDisbursementUseCase(
	repositories DeleteDisbursementRepositories,
	services DeleteDisbursementServices,
) *DeleteDisbursementUseCase {
	return &DeleteDisbursementUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete disbursement operation
func (uc *DeleteDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.DeleteDisbursementRequest) (*disbursementpb.DeleteDisbursementResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityDisbursement,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	return uc.repositories.Disbursement.DeleteDisbursement(ctx, req)
}
