package criteria_option

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type ReadCriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type ReadCriteriaOptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadCriteriaOptionUseCase handles the business logic for reading criteria options
type ReadCriteriaOptionUseCase struct {
	repositories ReadCriteriaOptionRepositories
	services     ReadCriteriaOptionServices
}

// NewReadCriteriaOptionUseCase creates a new ReadCriteriaOptionUseCase
func NewReadCriteriaOptionUseCase(
	repositories ReadCriteriaOptionRepositories,
	services ReadCriteriaOptionServices,
) *ReadCriteriaOptionUseCase {
	return &ReadCriteriaOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read criteria option operation
func (uc *ReadCriteriaOptionUseCase) Execute(ctx context.Context, req *pb.ReadCriteriaOptionRequest) (*pb.ReadCriteriaOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCriteriaOption, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes reading within a transaction
func (uc *ReadCriteriaOptionUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadCriteriaOptionRequest) (*pb.ReadCriteriaOptionResponse, error) {
	var result *pb.ReadCriteriaOptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for reading a criteria option
func (uc *ReadCriteriaOptionUseCase) executeCore(ctx context.Context, req *pb.ReadCriteriaOptionRequest) (*pb.ReadCriteriaOptionResponse, error) {
	resp, err := uc.repositories.CriteriaOption.ReadCriteriaOption(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.errors.not_found", "[ERR-DEFAULT] Criteria option not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.errors.not_found", "[ERR-DEFAULT] Criteria option not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCriteriaOptionUseCase) validateInput(ctx context.Context, req *pb.ReadCriteriaOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "criteria_option.validation.id_required", "[ERR-DEFAULT] Criteria option ID is required"))
	}
	return nil
}
