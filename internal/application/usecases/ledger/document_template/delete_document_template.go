package document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// DeleteDocumentTemplateRepositories groups all repository dependencies
type DeleteDocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// DeleteDocumentTemplateServices groups all business service dependencies
type DeleteDocumentTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDocumentTemplate, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "document_template.validation.id_required", "Document template ID is required [DEFAULT]"))
	}

	return uc.repositories.DocumentTemplate.DeleteDocumentTemplate(ctx, req)
}
