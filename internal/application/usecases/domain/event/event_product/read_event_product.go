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

// ReadEventProductRepositories groups all repository dependencies
type ReadEventProductRepositories struct {
	EventProduct eventproductpb.EventProductDomainServiceServer // Primary entity repository
	Event        eventpb.EventDomainServiceServer               // Entity reference validation
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// ReadEventProductServices groups all business service dependencies
type ReadEventProductServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return &ReadEventProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read event product operation
func (uc *ReadEventProductUseCase) Execute(ctx context.Context, req *eventproductpb.ReadEventProductRequest) (*eventproductpb.ReadEventProductResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventProduct,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.EventProduct, entityid.ActionRead)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "event_product.errors.authorization_failed", "Authorization failed for event product")
		return nil, errors.New(translatedError)
	}

	// Call repository
	return uc.repositories.EventProduct.ReadEventProduct(ctx, req)
}
