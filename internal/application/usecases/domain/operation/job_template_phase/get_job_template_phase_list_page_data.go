package job_template_phase

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type GetJobTemplatePhaseListPageDataRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type GetJobTemplatePhaseListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetJobTemplatePhaseListPageDataUseCase handles the business logic for getting job template phase list page data
type GetJobTemplatePhaseListPageDataUseCase struct {
	repositories GetJobTemplatePhaseListPageDataRepositories
	services     GetJobTemplatePhaseListPageDataServices
}

// NewGetJobTemplatePhaseListPageDataUseCase creates a new GetJobTemplatePhaseListPageDataUseCase
func NewGetJobTemplatePhaseListPageDataUseCase(
	repositories GetJobTemplatePhaseListPageDataRepositories,
	services GetJobTemplatePhaseListPageDataServices,
) *GetJobTemplatePhaseListPageDataUseCase {
	return &GetJobTemplatePhaseListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get job template phase list page data operation
func (uc *GetJobTemplatePhaseListPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplatePhase,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list page data retrieval within a transaction
func (uc *GetJobTemplatePhaseListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {
	var result *pb.GetJobTemplatePhaseListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"job_template_phase.errors.list_page_data_failed",
				"job template phase list page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting job template phase list page data
func (uc *GetJobTemplatePhaseListPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {
	resp, err := uc.repositories.JobTemplatePhase.GetJobTemplatePhaseListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"job_template_phase.errors.list_page_data_failed",
			"failed to retrieve job template phase list page data: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *GetJobTemplatePhaseListPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"job_template_phase.validation.request_required",
			"request is required",
		))
	}

	// Validate pagination if provided
	if req.Pagination != nil {
		if err := uc.validatePagination(ctx, req.Pagination); err != nil {
			return err
		}
	}

	return nil
}

// validatePagination validates pagination parameters
func (uc *GetJobTemplatePhaseListPageDataUseCase) validatePagination(
	ctx context.Context,
	pagination *commonpb.PaginationRequest,
) error {
	if pagination.Limit < 0 || pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"job_template_phase.validation.invalid_limit",
			"pagination limit must be between 1 and 100",
		))
	}

	return nil
}
