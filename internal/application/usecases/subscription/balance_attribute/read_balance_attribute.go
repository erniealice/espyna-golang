package balance_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// ReadBalanceAttributeRepositories groups all repository dependencies
type ReadBalanceAttributeRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
}

// ReadBalanceAttributeServices groups all business service dependencies
type ReadBalanceAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadBalanceAttributeUseCase handles the business logic for reading balance attributes
type ReadBalanceAttributeUseCase struct {
	repositories ReadBalanceAttributeRepositories
	services     ReadBalanceAttributeServices
}

// NewReadBalanceAttributeUseCase creates a new ReadBalanceAttributeUseCase
func NewReadBalanceAttributeUseCase(
	repositories ReadBalanceAttributeRepositories,
	services ReadBalanceAttributeServices,
) *ReadBalanceAttributeUseCase {
	return &ReadBalanceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read balance attribute operation
func (uc *ReadBalanceAttributeUseCase) Execute(ctx context.Context, req *balanceattributepb.ReadBalanceAttributeRequest) (*balanceattributepb.ReadBalanceAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalanceAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.ReadBalanceAttribute(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("balance_attribute with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"balance_attribute.errors.not_found",
				map[string]interface{}{"balanceAttributeId": req.Data.Id},
				"Balance attribute not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadBalanceAttributeUseCase) validateInput(ctx context.Context, req *balanceattributepb.ReadBalanceAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.id_required", "Balance attribute ID is required [DEFAULT]"))
	}
	return nil
}
