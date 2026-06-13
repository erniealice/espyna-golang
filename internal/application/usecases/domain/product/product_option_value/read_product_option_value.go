package product_option_value

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// ReadProductOptionValueRepositories groups all repository dependencies
type ReadProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// ReadProductOptionValueServices groups all business service dependencies
type ReadProductOptionValueServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Transaction management
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadProductOptionValueUseCase handles the business logic for reading a product option value
type ReadProductOptionValueUseCase struct {
	repositories ReadProductOptionValueRepositories
	services     ReadProductOptionValueServices
}

// NewReadProductOptionValueUseCase creates use case with grouped dependencies
func NewReadProductOptionValueUseCase(
	repositories ReadProductOptionValueRepositories,
	services ReadProductOptionValueServices,
) *ReadProductOptionValueUseCase {
	return &ReadProductOptionValueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product option value operation
func (uc *ReadProductOptionValueUseCase) Execute(ctx context.Context, req *productoptionvaluepb.ReadProductOptionValueRequest) (*productoptionvaluepb.ReadProductOptionValueResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductOptionValue,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.ProductOptionValue, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductOptionValue.ReadProductOptionValue(ctx, req)
	if err != nil {
		// Handle not found errors by checking for specific patterns in error message
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.not_found", "Product option value not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.errors.read_failed", "Failed to retrieve product option value [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductOptionValueUseCase) validateInput(ctx context.Context, req *productoptionvaluepb.ReadProductOptionValueRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.validation.data_required", "Product option value data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_option_value.validation.id_required", "Product option value ID is required [DEFAULT]"))
	}
	return nil
}
