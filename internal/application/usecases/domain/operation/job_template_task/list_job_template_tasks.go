package job_template_task

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

type ListJobTemplateTasksRepositories struct {
	JobTemplateTask pb.JobTemplateTaskDomainServiceServer
}

type ListJobTemplateTasksServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListJobTemplateTasksUseCase handles the business logic for listing job template tasks
type ListJobTemplateTasksUseCase struct {
	repositories ListJobTemplateTasksRepositories
	services     ListJobTemplateTasksServices
}

// NewListJobTemplateTasksUseCase creates a new ListJobTemplateTasksUseCase
func NewListJobTemplateTasksUseCase(
	repositories ListJobTemplateTasksRepositories,
	services ListJobTemplateTasksServices,
) *ListJobTemplateTasksUseCase {
	return &ListJobTemplateTasksUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list job template tasks operation
func (uc *ListJobTemplateTasksUseCase) Execute(ctx context.Context, req *pb.ListJobTemplateTasksRequest) (*pb.ListJobTemplateTasksResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplateTask,
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

// executeWithTransaction executes listing within a transaction
func (uc *ListJobTemplateTasksUseCase) executeWithTransaction(ctx context.Context, req *pb.ListJobTemplateTasksRequest) (*pb.ListJobTemplateTasksResponse, error) {
	var result *pb.ListJobTemplateTasksResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "job_template_task.errors.list_failed", "job template task listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing job template tasks
func (uc *ListJobTemplateTasksUseCase) executeCore(ctx context.Context, req *pb.ListJobTemplateTasksRequest) (*pb.ListJobTemplateTasksResponse, error) {
	resp, err := uc.repositories.JobTemplateTask.ListJobTemplateTasks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_task.errors.list_failed", "job template task listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListJobTemplateTasksUseCase) validateInput(ctx context.Context, req *pb.ListJobTemplateTasksRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_task.validation.request_required", "request is required"))
	}

	return nil
}
