package template_task_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type GetTemplateTaskCriteriaItemPageDataRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type GetTemplateTaskCriteriaItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetTemplateTaskCriteriaItemPageDataUseCase handles the business logic for getting template task criteria item page data
type GetTemplateTaskCriteriaItemPageDataUseCase struct {
	repositories GetTemplateTaskCriteriaItemPageDataRepositories
	services     GetTemplateTaskCriteriaItemPageDataServices
}

// NewGetTemplateTaskCriteriaItemPageDataUseCase creates a new GetTemplateTaskCriteriaItemPageDataUseCase
func NewGetTemplateTaskCriteriaItemPageDataUseCase(
	repositories GetTemplateTaskCriteriaItemPageDataRepositories,
	services GetTemplateTaskCriteriaItemPageDataServices,
) *GetTemplateTaskCriteriaItemPageDataUseCase {
	return &GetTemplateTaskCriteriaItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get template task criteria item page data operation
func (uc *GetTemplateTaskCriteriaItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaItemPageDataRequest,
) (*pb.GetTemplateTaskCriteriaItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.TemplateTaskCriteria, entityid.ActionList); err != nil {
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

// executeWithTransaction executes item page data retrieval within a transaction
func (uc *GetTemplateTaskCriteriaItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaItemPageDataRequest,
) (*pb.GetTemplateTaskCriteriaItemPageDataResponse, error) {
	var result *pb.GetTemplateTaskCriteriaItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"template_task_criteria.errors.item_page_data_failed",
				"template task criteria item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting template task criteria item page data
func (uc *GetTemplateTaskCriteriaItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaItemPageDataRequest,
) (*pb.GetTemplateTaskCriteriaItemPageDataResponse, error) {
	readReq := &pb.ReadTemplateTaskCriteriaRequest{
		Data: &pb.TemplateTaskCriteria{
			Id: req.TemplateTaskCriteriaId,
		},
	}

	readResp, err := uc.repositories.TemplateTaskCriteria.ReadTemplateTaskCriteria(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"template_task_criteria.errors.read_failed",
			"failed to retrieve template task criteria: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"template_task_criteria.errors.not_found",
			"template task criteria not found",
		))
	}

	item := readResp.Data[0]

	return &pb.GetTemplateTaskCriteriaItemPageDataResponse{
		TemplateTaskCriteria: item,
		Success:              true,
	}, nil
}

// validateInput validates the input request
func (uc *GetTemplateTaskCriteriaItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pb.GetTemplateTaskCriteriaItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"template_task_criteria.validation.request_required",
			"request is required",
		))
	}

	if req.TemplateTaskCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"template_task_criteria.validation.id_required",
			"template task criteria ID is required",
		))
	}

	return nil
}
