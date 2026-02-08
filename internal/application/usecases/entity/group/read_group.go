package group

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// ReadGroupRepositories groups all repository dependencies
type ReadGroupRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// ReadGroupServices groups all business service dependencies
type ReadGroupServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadGroupUseCase handles the business logic for reading groups
type ReadGroupUseCase struct {
	repositories ReadGroupRepositories
	services     ReadGroupServices
}

// NewReadGroupUseCase creates use case with grouped dependencies
func NewReadGroupUseCase(
	repositories ReadGroupRepositories,
	services ReadGroupServices,
) *ReadGroupUseCase {
	return &ReadGroupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadGroupUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadGroupUseCase with grouped parameters instead
func NewReadGroupUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *ReadGroupUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadGroupRepositories{
		Group: groupRepo,
	}

	services := ReadGroupServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadGroupUseCase(repositories, services)
}

func (uc *ReadGroupUseCase) Execute(ctx context.Context, req *grouppb.ReadGroupRequest) (*grouppb.ReadGroupResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Group.ReadGroup(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadGroupUseCase) validateInput(ctx context.Context, req *grouppb.ReadGroupRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.data_required", "Group data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.id_required", "Group ID is required [DEFAULT]"))
	}
	return nil
}
