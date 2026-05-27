package disbursementmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// ListDisbursementMethodsRepositories groups all repository dependencies.
type ListDisbursementMethodsRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// ListDisbursementMethodsServices groups all business service dependencies.
type ListDisbursementMethodsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListDisbursementMethodsUseCase handles the business logic for listing disbursement methods.
type ListDisbursementMethodsUseCase struct {
	repositories ListDisbursementMethodsRepositories
	services     ListDisbursementMethodsServices
}

// NewListDisbursementMethodsUseCase creates a new ListDisbursementMethodsUseCase.
func NewListDisbursementMethodsUseCase(
	repositories ListDisbursementMethodsRepositories,
	services ListDisbursementMethodsServices,
) *ListDisbursementMethodsUseCase {
	return &ListDisbursementMethodsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list disbursement methods operation.
func (uc *ListDisbursementMethodsUseCase) Execute(ctx context.Context, req *disbursementmethodpb.ListDisbursementMethodsRequest) (*disbursementmethodpb.ListDisbursementMethodsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	return uc.repositories.DisbursementMethod.ListDisbursementMethods(ctx, req)
}
