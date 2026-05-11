package product_line

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
)

// CreateProductLineUseCase handles the business logic for creating product lines
// CreateProductLineRepositories groups all repository dependencies
type CreateProductLineRepositories struct {
	ProductLine productlinepb.ProductLineDomainServiceServer // Primary entity repository
	Product     productpb.ProductDomainServiceServer
	Line        linepb.LineDomainServiceServer
}

// CreateProductLineServices groups all business service dependencies
type CreateProductLineServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductLineUseCase handles the business logic for creating product lines
type CreateProductLineUseCase struct {
	repositories CreateProductLineRepositories
	services     CreateProductLineServices
}

// NewCreateProductLineUseCase creates a new CreateProductLineUseCase
func NewCreateProductLineUseCase(
	repositories CreateProductLineRepositories,
	services CreateProductLineServices,
) *CreateProductLineUseCase {
	return &CreateProductLineUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product line operation
func (uc *CreateProductLineUseCase) Execute(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductLine, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product line creation within a transaction
func (uc *CreateProductLineUseCase) executeWithTransaction(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
	var result *productlinepb.CreateProductLineResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_line.errors.creation_failed", "Product line creation failed [DEFAULT]")
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
func (uc *CreateProductLineUseCase) executeCore(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductLine, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.authorization_failed", "Authorization failed for product lines [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichProductLineData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductLine.CreateProductLine(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.creation_failed", "Product line creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreateProductLineUseCase) validateInput(ctx context.Context, req *productlinepb.CreateProductLineRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.data_required", "Product line data is required [DEFAULT]"))
	}
	// ProductLine doesn't have Name field - removed invalid check
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.LineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.line_id_required", "Line ID is required [DEFAULT]"))
	}
	return nil
}

// enrichProductLineData adds generated fields and audit information
func (uc *CreateProductLineUseCase) enrichProductLineData(productLine *productlinepb.ProductLine) error {
	now := time.Now()

	// Generate ProductLine ID if not provided
	if productLine.Id == "" {
		productLine.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	productLine.DateCreated = &[]int64{now.UnixMilli()}[0]
	productLine.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	productLine.DateModified = &[]int64{now.UnixMilli()}[0]
	productLine.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	productLine.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for product lines
func (uc *CreateProductLineUseCase) validateBusinessRules(ctx context.Context, productLine *productlinepb.ProductLine) error {
	// Validate product ID format
	if len(productLine.ProductId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate line ID format
	if len(productLine.LineId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.line_id_min_length", "Line ID must be at least 2 characters long [DEFAULT]"))
	}

	// Business constraint: Product line must be associated with valid product and line
	if productLine.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.product_association_required", "Product line must be associated with a product [DEFAULT]"))
	}

	if productLine.LineId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.validation.line_association_required", "Product line must be associated with a line [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateProductLineUseCase) validateEntityReferences(ctx context.Context, productLine *productlinepb.ProductLine) error {
	// Validate Product entity reference
	if productLine.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productLine.ProductId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", productLine.ProductId)
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.line_reference_validation_failed", "Failed to validate line entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if line == nil || line.Data == nil || len(line.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.line_not_found", "Referenced line with ID '{lineId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{lineId}", productLine.LineId)
			return errors.New(translatedError)
		}
		if !line.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_line.errors.line_not_active", "Referenced line with ID '{lineId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{lineId}", productLine.LineId)
			return errors.New(translatedError)
		}
	}

	return nil
}
