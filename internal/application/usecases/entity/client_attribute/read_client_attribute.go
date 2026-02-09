package client_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// ReadClientAttributeUseCase handles the business logic for reading client attributes
// ReadClientAttributeRepositories groups all repository dependencies
type ReadClientAttributeRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
}

// ReadClientAttributeServices groups all business service dependencies
type ReadClientAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadClientAttributeUseCase handles the business logic for reading client attributes
type ReadClientAttributeUseCase struct {
	repositories ReadClientAttributeRepositories
	services     ReadClientAttributeServices
}

// NewReadClientAttributeUseCase creates use case with grouped dependencies
func NewReadClientAttributeUseCase(
	repositories ReadClientAttributeRepositories,
	services ReadClientAttributeServices,
) *ReadClientAttributeUseCase {
	return &ReadClientAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadClientAttributeUseCaseUngrouped creates a new ReadClientAttributeUseCase
// Deprecated: Use NewReadClientAttributeUseCase with grouped parameters instead
func NewReadClientAttributeUseCaseUngrouped(clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer) *ReadClientAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
	}

	services := ReadClientAttributeServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadClientAttributeUseCase(repositories, services)
}

// Execute performs the read client attribute operation
func (uc *ReadClientAttributeUseCase) Execute(ctx context.Context, req *clientattributepb.ReadClientAttributeRequest) (*clientattributepb.ReadClientAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityClientAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.ClientAttribute.ReadClientAttribute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadClientAttributeUseCase) validateInput(ctx context.Context, req *clientattributepb.ReadClientAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client_attribute.validation.id_required", ""))
	}
	return nil
}
