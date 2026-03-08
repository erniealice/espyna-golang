package disbursement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// DeleteDisbursementRepositories groups all repository dependencies
type DeleteDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// DeleteDisbursementServices groups all business service dependencies
type DeleteDisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDisbursement, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	return uc.repositories.Disbursement.DeleteDisbursement(ctx, req)
}
