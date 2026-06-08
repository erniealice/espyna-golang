package eventproduct

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// DeleteEventProductRepositories groups all repository dependencies
type DeleteEventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// DeleteEventProductServices groups all business service dependencies
type DeleteEventProductServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// DeleteEventProductUseCase handles the business logic for deleting event product associations
type DeleteEventProductUseCase struct {
	repositories DeleteEventProductRepositories
	services     DeleteEventProductServices
}

// NewDeleteEventProductUseCase creates a new DeleteEventProductUseCase
func NewDeleteEventProductUseCase(
	repositories DeleteEventProductRepositories,
	services DeleteEventProductServices,
) *DeleteEventProductUseCase {
	return &DeleteEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteEventProductUseCaseUngrouped creates a new DeleteEventProductUseCase
// Deprecated: Use NewDeleteEventProductUseCase with grouped parameters instead
func NewDeleteEventProductUseCaseUngrouped(eventProductRepo eventproductpb.EventProductDomainServiceServer) *DeleteEventProductUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteEventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        nil,
		Product:      nil,
	}

	services := DeleteEventProductServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &DeleteEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete event product operation
func (uc *DeleteEventProductUseCase) Execute(ctx context.Context, req *eventproductpb.DeleteEventProductRequest) (*eventproductpb.DeleteEventProductResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.EventProduct, entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventProduct, entityid.ActionDelete)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventProduct.DeleteEventProduct(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteEventProductUseCase) validateInput(req *eventproductpb.DeleteEventProductRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event product data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event product ID is required")
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteEventProductUseCase) validateBusinessRules(eventProduct *eventproductpb.EventProduct) error {
	// Additional business rules can be added here:
	// - Check if event product association can be safely deleted
	// - Validate impact on event pricing/totals
	// - Check for related records that might be affected

	return nil
}
