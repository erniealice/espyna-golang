package job_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// ReadJobTemplateRepositories groups all repository dependencies
type ReadJobTemplateRepositories struct {
	JobTemplate pb.JobTemplateDomainServiceServer
}

// ReadJobTemplateServices groups all business service dependencies
type ReadJobTemplateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadJobTemplateUseCase handles the business logic for reading job templates
type ReadJobTemplateUseCase struct {
	repositories ReadJobTemplateRepositories
	services     ReadJobTemplateServices
}

// NewReadJobTemplateUseCase creates a new ReadJobTemplateUseCase
func NewReadJobTemplateUseCase(
	repositories ReadJobTemplateRepositories,
	services ReadJobTemplateServices,
) *ReadJobTemplateUseCase {
	return &ReadJobTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read job template operation
func (uc *ReadJobTemplateUseCase) Execute(ctx context.Context, req *pb.ReadJobTemplateRequest) (*pb.ReadJobTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"job_template", entityid.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.JobTemplate.ReadJobTemplate(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.errors.not_found", "job template not found [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *ReadJobTemplateUseCase) validateInput(ctx context.Context, req *pb.ReadJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.request_required", "request is required"))
	}
	if req.Data == nil || req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template.validation.id_required", "job template ID is required"))
	}
	return nil
}
