package criteria_option

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/criteria_option"
)

type CreateCriteriaOptionRepositories struct {
	CriteriaOption pb.CriteriaOptionDomainServiceServer
}

type CreateCriteriaOptionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateCriteriaOptionUseCase handles the business logic for creating criteria options
type CreateCriteriaOptionUseCase struct {
	repositories CreateCriteriaOptionRepositories
	services     CreateCriteriaOptionServices
}

// NewCreateCriteriaOptionUseCase creates a new CreateCriteriaOptionUseCase
func NewCreateCriteriaOptionUseCase(
	repositories CreateCriteriaOptionRepositories,
	services CreateCriteriaOptionServices,
) *CreateCriteriaOptionUseCase {
	return &CreateCriteriaOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create criteria option operation
func (uc *CreateCriteriaOptionUseCase) Execute(ctx context.Context, req *pb.CreateCriteriaOptionRequest) (*pb.CreateCriteriaOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCriteriaOption, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateCriteriaOptionUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateCriteriaOptionRequest, enrichedData *pb.CriteriaOption) (*pb.CreateCriteriaOptionResponse, error) {
	var result *pb.CreateCriteriaOptionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedData)
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

// executeCore contains the core business logic for creating a criteria option
func (uc *CreateCriteriaOptionUseCase) executeCore(ctx context.Context, req *pb.CreateCriteriaOptionRequest, enrichedData *pb.CriteriaOption) (*pb.CreateCriteriaOptionResponse, error) {
	resp, err := uc.repositories.CriteriaOption.CreateCriteriaOption(ctx, &pb.CreateCriteriaOptionRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.errors.creation_failed", "[ERR-DEFAULT] Criteria option creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateCriteriaOptionUseCase) applyBusinessLogic(data *pb.CriteriaOption) *pb.CriteriaOption {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateCriteriaOptionUseCase) validateBusinessRules(ctx context.Context, data *pb.CriteriaOption) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.data_required", "[ERR-DEFAULT] Criteria option data is required"))
	}
	if data.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.criteria_id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	if data.OptionLabel == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "criteria_option.validation.label_required", "[ERR-DEFAULT] Criteria option label is required"))
	}

	return nil
}
