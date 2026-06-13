package eventproduct

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	eventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event"
	eventproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// ListEventProductsRepositories groups all repository dependencies
type ListEventProductsRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// ListEventProductsServices groups all business service dependencies
type ListEventProductsServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListEventProductsUseCase handles the business logic for listing event product associations
type ListEventProductsUseCase struct {
	repositories ListEventProductsRepositories
	services     ListEventProductsServices
}

// NewListEventProductsUseCase creates a new ListEventProductsUseCase
func NewListEventProductsUseCase(
	repositories ListEventProductsRepositories,
	services ListEventProductsServices,
) *ListEventProductsUseCase {
	return &ListEventProductsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListEventProductsUseCaseUngrouped creates a new ListEventProductsUseCase
// Deprecated: Use NewListEventProductsUseCase with grouped parameters instead
func NewListEventProductsUseCaseUngrouped(eventProductRepo eventproductpb.EventProductDomainServiceServer) *ListEventProductsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListEventProductsRepositories{
		EventProduct: eventProductRepo,
		Event:        nil,
		Product:      nil,
	}

	services := ListEventProductsServices{
		Authorizer: nil, // Will be injected later if needed
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ListEventProductsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list event products operation
func (uc *ListEventProductsUseCase) Execute(ctx context.Context, req *eventproductpb.ListEventProductsRequest) (*eventproductpb.ListEventProductsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventProduct,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventProduct, entityid.ActionList)
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

	// Handle nil request by creating default empty request for list operations
	if req == nil {
		req = &eventproductpb.ListEventProductsRequest{}
	}

	// Call repository
	return uc.repositories.EventProduct.ListEventProducts(ctx, req)
}

// validateInput validates the input request
func (uc *ListEventProductsUseCase) validateInput(req *eventproductpb.ListEventProductsRequest) error {
	// For list operations, nil request is allowed - we'll create a default empty request
	return nil
}
