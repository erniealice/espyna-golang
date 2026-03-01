package group_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	groupattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group_attribute"
)

// ReadGroupAttributeUseCase handles the business logic for reading group attributes
// ReadGroupAttributeRepositories groups all repository dependencies
type ReadGroupAttributeRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
}

// ReadGroupAttributeServices groups all business service dependencies
type ReadGroupAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadGroupAttributeUseCase handles the business logic for reading group attributes
type ReadGroupAttributeUseCase struct {
	repositories ReadGroupAttributeRepositories
	services     ReadGroupAttributeServices
}

// NewReadGroupAttributeUseCase creates use case with grouped dependencies
func NewReadGroupAttributeUseCase(
	repositories ReadGroupAttributeRepositories,
	services ReadGroupAttributeServices,
) *ReadGroupAttributeUseCase {
	return &ReadGroupAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadGroupAttributeUseCaseUngrouped creates a new ReadGroupAttributeUseCase
// Deprecated: Use NewReadGroupAttributeUseCase with grouped parameters instead
func NewReadGroupAttributeUseCaseUngrouped(groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer) *ReadGroupAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadGroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
	}

	services := ReadGroupAttributeServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewReadGroupAttributeUseCase(repositories, services)
}

// Execute performs the read group attribute operation
func (uc *ReadGroupAttributeUseCase) Execute(ctx context.Context, req *groupattributepb.ReadGroupAttributeRequest) (*groupattributepb.ReadGroupAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityGroupAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.ReadGroupAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadGroupAttributeUseCase) validateInput(ctx context.Context, req *groupattributepb.ReadGroupAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.data_required", "[ERR-DEFAULT] Group attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.id_required", "[ERR-DEFAULT] Attribute ID is required"))
	}
	return nil
}
