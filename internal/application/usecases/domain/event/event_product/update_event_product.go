package eventproduct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UpdateEventProductRepositories groups all repository dependencies
type UpdateEventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// UpdateEventProductServices groups all business service dependencies
type UpdateEventProductServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateEventProductUseCase handles the business logic for updating event product associations
type UpdateEventProductUseCase struct {
	repositories UpdateEventProductRepositories
	services     UpdateEventProductServices
}

// NewUpdateEventProductUseCase creates a new UpdateEventProductUseCase
func NewUpdateEventProductUseCase(
	repositories UpdateEventProductRepositories,
	services UpdateEventProductServices,
) *UpdateEventProductUseCase {
	return &UpdateEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateEventProductUseCaseUngrouped creates a new UpdateEventProductUseCase
// Deprecated: Use NewUpdateEventProductUseCase with grouped parameters instead
func NewUpdateEventProductUseCaseUngrouped(
	eventProductRepo eventproductpb.EventProductDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	productRepo productpb.ProductDomainServiceServer,
) *UpdateEventProductUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateEventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        eventRepo,
		Product:      productRepo,
	}

	services := UpdateEventProductServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &UpdateEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update event product operation
func (uc *UpdateEventProductUseCase) Execute(ctx context.Context, req *eventproductpb.UpdateEventProductRequest) (*eventproductpb.UpdateEventProductResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventProduct,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventProduct, entityid.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichEventProductData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.EventProduct.UpdateEventProduct(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateEventProductUseCase) validateInput(req *eventproductpb.UpdateEventProductRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event product data is required")
	}
	if req.Data.Id == "" {
		return errors.New("event product ID is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ProductId == "" {
		return errors.New("product ID is required")
	}
	return nil
}

// enrichEventProductData adds audit information for updates
func (uc *UpdateEventProductUseCase) enrichEventProductData(eventProduct *eventproductpb.EventProduct) error {
	now := time.Now()

	// Update audit fields
	eventProduct.DateModified = &[]int64{now.UnixMilli()}[0]
	eventProduct.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateEventProductUseCase) validateBusinessRules(eventProduct *eventproductpb.EventProduct) error {
	// Validate that event and product IDs are not the same
	if eventProduct.EventId == eventProduct.ProductId {
		return errors.New("event ID and product ID cannot be the same")
	}

	// Additional business rules can be added here:
	// - Validate pricing consistency (unit_price * quantity = total_price)
	// - Validate currency format
	// - Check product availability for the updated event

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateEventProductUseCase) validateEntityReferences(ctx context.Context, eventProduct *eventproductpb.EventProduct) error {
	// Validate Event entity reference
	if eventProduct.EventId != "" {
		event, err := uc.repositories.Event.ReadEvent(ctx, &eventpb.ReadEventRequest{
			Data: &eventpb.Event{Id: eventProduct.EventId},
		})
		if err != nil {
			return err
		}
		if event == nil || event.Data == nil || len(event.Data) == 0 {
			return fmt.Errorf("referenced event with ID '%s' does not exist", eventProduct.EventId)
		}
		if !event.Data[0].Active {
			return fmt.Errorf("referenced event with ID '%s' is not active", eventProduct.EventId)
		}
	}

	// Validate Product entity reference
	if eventProduct.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: eventProduct.ProductId},
		})
		if err != nil {
			return err
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			return fmt.Errorf("referenced product with ID '%s' does not exist", eventProduct.ProductId)
		}
		if !product.Data[0].Active {
			return fmt.Errorf("referenced product with ID '%s' is not active", eventProduct.ProductId)
		}
	}

	return nil
}
