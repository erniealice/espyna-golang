package eventproduct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"

	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// CreateEventProductRepositories groups all repository dependencies
type CreateEventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// CreateEventProductServices groups all business service dependencies
type CreateEventProductServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateEventProductUseCase handles the business logic for creating event product associations
type CreateEventProductUseCase struct {
	repositories CreateEventProductRepositories
	services     CreateEventProductServices
}

// NewCreateEventProductUseCase creates use case with grouped dependencies
func NewCreateEventProductUseCase(
	repositories CreateEventProductRepositories,
	services CreateEventProductServices,
) *CreateEventProductUseCase {
	return &CreateEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateEventProductUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateEventProductUseCase with grouped parameters instead
func NewCreateEventProductUseCaseUngrouped(
	eventProductRepo eventproductpb.EventProductDomainServiceServer,
	eventRepo eventpb.EventDomainServiceServer,
	productRepo productpb.ProductDomainServiceServer,
) *CreateEventProductUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateEventProductRepositories{
		EventProduct: eventProductRepo,
		Event:        eventRepo,
		Product:      productRepo,
	}

	services := CreateEventProductServices{
		Authorizer:  nil, // Will be injected later if needed
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return &CreateEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event product operation
func (uc *CreateEventProductUseCase) Execute(ctx context.Context, req *eventproductpb.CreateEventProductRequest) (*eventproductpb.CreateEventProductResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventProduct,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichEventProductData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *CreateEventProductUseCase) shouldUseTransaction(ctx context.Context) bool {
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return false
	}

	// Don't start a nested transaction if we're already in one
	if uc.services.Transactor.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreateEventProductUseCase) executeWithTransaction(ctx context.Context, req *eventproductpb.CreateEventProductRequest) (*eventproductpb.CreateEventProductResponse, error) {
	var response *eventproductpb.CreateEventProductResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Business rule validation (check first to avoid unnecessary DB calls)
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			return err
		}

		// Create EventProduct (will participate in transaction)
		createResponse, err := uc.repositories.EventProduct.CreateEventProduct(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event product: %w", err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction (backward compatibility)
func (uc *CreateEventProductUseCase) executeWithoutTransaction(ctx context.Context, req *eventproductpb.CreateEventProductRequest) (*eventproductpb.CreateEventProductResponse, error) {
	// Business rule validation (check first to avoid unnecessary DB calls)
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf("entity reference validation failed: %w", err)
	}

	// Call repository (no transaction)
	return uc.repositories.EventProduct.CreateEventProduct(ctx, req)
}

// validateInput validates the input request
func (uc *CreateEventProductUseCase) validateInput(req *eventproductpb.CreateEventProductRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event product data is required")
	}
	if req.Data.EventId == "" {
		return errors.New("event ID is required")
	}
	if req.Data.ProductId == "" {
		return errors.New("product ID is required")
	}
	return nil
}

// enrichEventProductData adds generated fields and audit information
func (uc *CreateEventProductUseCase) enrichEventProductData(eventProduct *eventproductpb.EventProduct) error {
	now := time.Now()

	// Generate EventProduct ID if not provided
	if eventProduct.Id == "" {
		eventProduct.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	eventProduct.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventProduct.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventProduct.DateModified = &[]int64{now.UnixMilli()}[0]
	eventProduct.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	eventProduct.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateEventProductUseCase) validateBusinessRules(eventProduct *eventproductpb.EventProduct) error {
	// Validate event and product IDs are not the same
	if eventProduct.EventId == eventProduct.ProductId {
		return errors.New("event ID and product ID cannot be the same")
	}

	// Additional business rules can be added here:
	// - Validate pricing consistency (unit_price * quantity = total_price)
	// - Validate currency format
	// - Check product availability for the event

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateEventProductUseCase) validateEntityReferences(ctx context.Context, eventProduct *eventproductpb.EventProduct) error {
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
