package eventproduct

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ReadEventProductRepositories groups all repository dependencies
type ReadEventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// ReadEventProductServices groups all business service dependencies
type ReadEventProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadEventProductUseCase handles the business logic for reading event product associations
type ReadEventProductUseCase struct {
	repositories ReadEventProductRepositories
	services     ReadEventProductServices
}

// NewReadEventProductUseCase creates use case with grouped dependencies
func NewReadEventProductUseCase(
	repositories ReadEventProductRepositories,
	services ReadEventProductServices,
) *ReadEventProductUseCase {
	return &ReadEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadEventProductUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadEventProductUseCase with grouped parameters instead
func NewReadEventProductUseCaseUngrouped(
	eventProductRepo eventproductpb.EventProductDomainServiceServer,
) *ReadEventProductUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadEventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        nil,
		Product:      nil,
	}

	services := ReadEventProductServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return &ReadEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event product operation
func (uc *ReadEventProductUseCase) Execute(ctx context.Context, req *eventproductpb.ReadEventProductRequest) (*eventproductpb.ReadEventProductResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityEventProduct, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityEventProduct, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	// Call repository
	return uc.repositories.EventProduct.ReadEventProduct(ctx, req)
}
