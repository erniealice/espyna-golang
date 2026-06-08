package product_line

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

// ListProductLinesUseCase handles the business logic for listing product lines
// ListProductLinesRepositories groups all repository dependencies
type ListProductLinesRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
}

// ListProductLinesServices groups all business service dependencies
type ListProductLinesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ListProductLinesUseCase handles the business logic for listing product lines
type ListProductLinesUseCase struct {
	repositories ListProductLinesRepositories
	services     ListProductLinesServices
}

// NewListProductLinesUseCase creates a new ListProductLinesUseCase
func NewListProductLinesUseCase(
	repositories ListProductLinesRepositories,
	services ListProductLinesServices,
) *ListProductLinesUseCase {
	return &ListProductLinesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product lines operation
func (uc *ListProductLinesUseCase) Execute(ctx context.Context, req *productlinepb.ListProductLinesRequest) (*productlinepb.ListProductLinesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.ProductLine, entityid.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductLine, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductLine.ListProductLines(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.list_failed", "Failed to retrieve product lines [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductLinesUseCase) validateInput(ctx context.Context, req *productlinepb.ListProductLinesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
