package resource

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// ReadResourceUseCase handles the business logic for reading a resource
// ReadResourceRepositories groups all repository dependencies
type ReadResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
}

// ReadResourceServices groups all business service dependencies
type ReadResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadResourceUseCase handles the business logic for reading a resource
type ReadResourceUseCase struct {
	repositories ReadResourceRepositories
	services     ReadResourceServices
}

// NewReadResourceUseCase creates a new ReadResourceUseCase
func NewReadResourceUseCase(
	repositories ReadResourceRepositories,
	services ReadResourceServices,
) *ReadResourceUseCase {
	return &ReadResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read resource operation
func (uc *ReadResourceUseCase) Execute(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityResource, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Resource.ReadResource(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.not_found", "Resource with ID \"{resourceId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{resourceId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadResourceUseCase) validateInput(ctx context.Context, req *resourcepb.ReadResourceRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.data_required", "Resource data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.id_required", "Resource ID is required [DEFAULT]"))
	}
	return nil
}
