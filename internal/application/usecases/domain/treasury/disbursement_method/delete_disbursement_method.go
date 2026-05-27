package disbursementmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// DeleteDisbursementMethodRepositories groups all repository dependencies.
type DeleteDisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// DeleteDisbursementMethodServices groups all business service dependencies.
type DeleteDisbursementMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteDisbursementMethodUseCase handles the business logic for deleting disbursement methods.
type DeleteDisbursementMethodUseCase struct {
	repositories DeleteDisbursementMethodRepositories
	services     DeleteDisbursementMethodServices
}

// NewDeleteDisbursementMethodUseCase creates a new DeleteDisbursementMethodUseCase.
func NewDeleteDisbursementMethodUseCase(
	repositories DeleteDisbursementMethodRepositories,
	services DeleteDisbursementMethodServices,
) *DeleteDisbursementMethodUseCase {
	return &DeleteDisbursementMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete disbursement method operation.
func (uc *DeleteDisbursementMethodUseCase) Execute(ctx context.Context, req *disbursementmethodpb.DeleteDisbursementMethodRequest) (*disbursementmethodpb.DeleteDisbursementMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.id_required", "Disbursement method ID is required [DEFAULT]"))
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	return uc.repositories.DisbursementMethod.DeleteDisbursementMethod(ctx, req)
}
