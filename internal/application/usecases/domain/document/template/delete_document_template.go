package template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
)

// DeleteDocumentTemplateRepositories groups all repository dependencies
type DeleteDocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// DeleteDocumentTemplateServices groups all business service dependencies
type DeleteDocumentTemplateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteDocumentTemplateUseCase handles the business logic for deleting document templates
type DeleteDocumentTemplateUseCase struct {
	repositories DeleteDocumentTemplateRepositories
	services     DeleteDocumentTemplateServices
}

// NewDeleteDocumentTemplateUseCase creates a new DeleteDocumentTemplateUseCase
func NewDeleteDocumentTemplateUseCase(
	repositories DeleteDocumentTemplateRepositories,
	services DeleteDocumentTemplateServices,
) *DeleteDocumentTemplateUseCase {
	return &DeleteDocumentTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete document template operation
func (uc *DeleteDocumentTemplateUseCase) Execute(ctx context.Context, req *documenttemplatepb.DeleteDocumentTemplateRequest) (*documenttemplatepb.DeleteDocumentTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDocumentTemplate, entityid.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "document_template.validation.id_required", "Document template ID is required [DEFAULT]"))
	}

	return uc.repositories.DocumentTemplate.DeleteDocumentTemplate(ctx, req)
}
