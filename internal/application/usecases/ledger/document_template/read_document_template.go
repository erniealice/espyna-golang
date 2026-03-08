package document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// ReadDocumentTemplateRepositories groups all repository dependencies
type ReadDocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// ReadDocumentTemplateServices groups all business service dependencies
type ReadDocumentTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadDocumentTemplateUseCase handles the business logic for reading a document template
type ReadDocumentTemplateUseCase struct {
	repositories ReadDocumentTemplateRepositories
	services     ReadDocumentTemplateServices
}

// NewReadDocumentTemplateUseCase creates use case with grouped dependencies
func NewReadDocumentTemplateUseCase(
	repositories ReadDocumentTemplateRepositories,
	services ReadDocumentTemplateServices,
) *ReadDocumentTemplateUseCase {
	return &ReadDocumentTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read document template operation
func (uc *ReadDocumentTemplateUseCase) Execute(ctx context.Context, req *documenttemplatepb.ReadDocumentTemplateRequest) (*documenttemplatepb.ReadDocumentTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDocumentTemplate, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "document_template.validation.id_required", "Document template ID is required [DEFAULT]"))
	}

	return uc.repositories.DocumentTemplate.ReadDocumentTemplate(ctx, req)
}
