package balance_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// ListBalanceAttributesRepositories groups all repository dependencies
type ListBalanceAttributesRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
}

// ListBalanceAttributesServices groups all business service dependencies
type ListBalanceAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListBalanceAttributesUseCase handles the business logic for listing balance attributes
type ListBalanceAttributesUseCase struct {
	repositories ListBalanceAttributesRepositories
	services     ListBalanceAttributesServices
}

// NewListBalanceAttributesUseCase creates a new ListBalanceAttributesUseCase
func NewListBalanceAttributesUseCase(
	repositories ListBalanceAttributesRepositories,
	services ListBalanceAttributesServices,
) *ListBalanceAttributesUseCase {
	return &ListBalanceAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list balance attributes operation
func (uc *ListBalanceAttributesUseCase) Execute(ctx context.Context, req *balanceattributepb.ListBalanceAttributesRequest) (*balanceattributepb.ListBalanceAttributesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalanceAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.ListBalanceAttributes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListBalanceAttributesUseCase) validateInput(ctx context.Context, req *balanceattributepb.ListBalanceAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
