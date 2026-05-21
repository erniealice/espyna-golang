package product_line

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

// UpdateProductLineUseCase handles the business logic for updating product lines
// UpdateProductLineRepositories groups all repository dependencies
type UpdateProductLineRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
	Product     productpb.ProductDomainServiceServer
	Line        linepb.LineDomainServiceServer
}

// UpdateProductLineServices groups all business service dependencies
type UpdateProductLineServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// UpdateProductLineUseCase handles the business logic for updating product lines
type UpdateProductLineUseCase struct {
	repositories UpdateProductLineRepositories
	services     UpdateProductLineServices
}

// NewUpdateProductLineUseCase creates a new UpdateProductLineUseCase
func NewUpdateProductLineUseCase(
	repositories UpdateProductLineRepositories,
	services UpdateProductLineServices,
) *UpdateProductLineUseCase {
	return &UpdateProductLineUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product line operation
func (uc *UpdateProductLineUseCase) Execute(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityProductLine, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product line update within a transaction
func (uc *UpdateProductLineUseCase) executeWithTransaction(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
	var result *productlinepb.UpdateProductLineResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "product_line.errors.update_failed", "Product line update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdateProductLineUseCase) executeCore(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductLine, ports.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichProductLineData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductLine.UpdateProductLine(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductLineUseCase) validateInput(ctx context.Context, req *productlinepb.UpdateProductLineRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.data_required", "Product line data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.id_required", "Product line ID is required [DEFAULT]"))
	}
	// ProductLine uses ProductId and LineId for identification
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.LineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.line_id_required", "Line ID is required [DEFAULT]"))
	}
	return nil
}

// enrichProductLineData adds generated fields and audit information
func (uc *UpdateProductLineUseCase) enrichProductLineData(productLine *productlinepb.ProductLine) error {
	now := time.Now()

	// Update audit fields
	productLine.DateModified = &[]int64{now.UnixMilli()}[0]
	productLine.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for product lines
func (uc *UpdateProductLineUseCase) validateBusinessRules(ctx context.Context, productLine *productlinepb.ProductLine) error {
	// Validate product ID format
	if len(productLine.ProductId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate line ID format
	if len(productLine.LineId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.line_id_min_length", "Line ID must be at least 2 characters long [DEFAULT]"))
	}

	// Business constraint: Product line must be associated with valid product and line
	if productLine.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.product_association_required", "Product line must be associated with a product [DEFAULT]"))
	}

	if productLine.LineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.validation.line_association_required", "Product line must be associated with a line [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateProductLineUseCase) validateEntityReferences(ctx context.Context, productLine *productlinepb.ProductLine) error {
	// Validate Product entity reference
	if productLine.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productLine.ProductId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", productLine.ProductId)
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", productLine.ProductId)
			return errors.New(translatedError)
		}
	}

	// Validate Line entity reference
	if productLine.LineId != "" {
		line, err := uc.repositories.Line.ReadLine(ctx, &linepb.ReadLineRequest{
			Data: &linepb.Line{Id: productLine.LineId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.line_reference_validation_failed", "Failed to validate line entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if line == nil || line.Data == nil || len(line.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.line_not_found", "Referenced line with ID '{lineId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{lineId}", productLine.LineId)
			return errors.New(translatedError)
		}
		if !line.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_line.errors.line_not_active", "Referenced line with ID '{lineId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{lineId}", productLine.LineId)
			return errors.New(translatedError)
		}
	}

	return nil
}
