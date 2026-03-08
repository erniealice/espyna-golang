package document_template

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// UpdateDocumentTemplateRepositories groups all repository dependencies
type UpdateDocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// UpdateDocumentTemplateServices groups all business service dependencies
type UpdateDocumentTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateDocumentTemplateUseCase handles the business logic for updating document templates
type UpdateDocumentTemplateUseCase struct {
	repositories UpdateDocumentTemplateRepositories
	services     UpdateDocumentTemplateServices
}

// NewUpdateDocumentTemplateUseCase creates use case with grouped dependencies
func NewUpdateDocumentTemplateUseCase(
	repositories UpdateDocumentTemplateRepositories,
	services UpdateDocumentTemplateServices,
) *UpdateDocumentTemplateUseCase {
	return &UpdateDocumentTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update document template operation
func (uc *UpdateDocumentTemplateUseCase) Execute(ctx context.Context, req *documenttemplatepb.UpdateDocumentTemplateRequest) (*documenttemplatepb.UpdateDocumentTemplateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDocumentTemplate, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *documenttemplatepb.UpdateDocumentTemplateResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("document template update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateDocumentTemplateUseCase) executeCore(ctx context.Context, req *documenttemplatepb.UpdateDocumentTemplateRequest) (*documenttemplatepb.UpdateDocumentTemplateResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "document_template.validation.id_required", "Document template ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.DocumentTemplate.UpdateDocumentTemplate(ctx, req)
}
