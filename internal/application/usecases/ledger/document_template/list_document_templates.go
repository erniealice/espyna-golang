package document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// ListDocumentTemplatesRepositories groups all repository dependencies
type ListDocumentTemplatesRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// ListDocumentTemplatesServices groups all business service dependencies
type ListDocumentTemplatesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListDocumentTemplatesUseCase handles the business logic for listing document templates
type ListDocumentTemplatesUseCase struct {
	repositories ListDocumentTemplatesRepositories
	services     ListDocumentTemplatesServices
}

// NewListDocumentTemplatesUseCase creates a new ListDocumentTemplatesUseCase
func NewListDocumentTemplatesUseCase(
	repositories ListDocumentTemplatesRepositories,
	services ListDocumentTemplatesServices,
) *ListDocumentTemplatesUseCase {
	return &ListDocumentTemplatesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list document templates operation
func (uc *ListDocumentTemplatesUseCase) Execute(ctx context.Context, req *documenttemplatepb.ListDocumentTemplatesRequest) (*documenttemplatepb.ListDocumentTemplatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDocumentTemplate, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "document_template.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.DocumentTemplate.ListDocumentTemplates(ctx, req)
}
