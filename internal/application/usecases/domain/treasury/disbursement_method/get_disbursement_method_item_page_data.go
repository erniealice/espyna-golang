package disbursementmethod

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// GetDisbursementMethodItemPageDataRepositories groups all repository dependencies.
type GetDisbursementMethodItemPageDataRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// GetDisbursementMethodItemPageDataServices groups all business service dependencies.
type GetDisbursementMethodItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetDisbursementMethodItemPageDataUseCase handles fetching a single enriched item.
type GetDisbursementMethodItemPageDataUseCase struct {
	repositories GetDisbursementMethodItemPageDataRepositories
	services     GetDisbursementMethodItemPageDataServices
}

// NewGetDisbursementMethodItemPageDataUseCase creates use case with grouped dependencies.
func NewGetDisbursementMethodItemPageDataUseCase(
	repositories GetDisbursementMethodItemPageDataRepositories,
	services GetDisbursementMethodItemPageDataServices,
) *GetDisbursementMethodItemPageDataUseCase {
	return &GetDisbursementMethodItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get disbursement method item page data operation.
func (uc *GetDisbursementMethodItemPageDataUseCase) Execute(ctx context.Context, req *disbursementmethodpb.GetDisbursementMethodItemPageDataRequest) (*disbursementmethodpb.GetDisbursementMethodItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.DisbursementMethodId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.id_required", "Disbursement method ID is required [DEFAULT]"))
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	resp, err := uc.repositories.DisbursementMethod.GetDisbursementMethodItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load disbursement method")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
