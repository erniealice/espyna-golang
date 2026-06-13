package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// ListJobTemplatesRepositories groups all repository dependencies
type ListJobTemplatesRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// ListJobTemplatesServices groups all business service dependencies
type ListJobTemplatesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListJobTemplatesUseCase handles the business logic for listing job templates
type ListJobTemplatesUseCase struct {
	repositories ListJobTemplatesRepositories
	services     ListJobTemplatesServices
}

// NewListJobTemplatesUseCase creates a new ListJobTemplatesUseCase
func NewListJobTemplatesUseCase(
	repositories ListJobTemplatesRepositories,
	services ListJobTemplatesServices,
) *ListJobTemplatesUseCase {
	return &ListJobTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list job templates operation
func (uc *ListJobTemplatesUseCase) Execute(ctx context.Context, req *pb.ListJobTemplatesRequest) (*pb.ListJobTemplatesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "job_template",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.request_required", "request is required"))
	}

	// Call repository
	result, err := uc.repositories.JobTemplate.ListJobTemplates(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.errors.list_failed", "job template listing failed [DEFAULT]"))
	}

	return result, nil
}
