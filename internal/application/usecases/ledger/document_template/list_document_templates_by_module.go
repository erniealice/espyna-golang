package document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// ListDocumentTemplatesByModuleRepositories groups all repository dependencies
type ListDocumentTemplatesByModuleRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// ListDocumentTemplatesByModuleServices groups all business service dependencies
type ListDocumentTemplatesByModuleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListDocumentTemplatesByModuleUseCase handles listing document templates filtered by module_key
type ListDocumentTemplatesByModuleUseCase struct {
	repositories ListDocumentTemplatesByModuleRepositories
	services     ListDocumentTemplatesByModuleServices
}

// NewListDocumentTemplatesByModuleUseCase creates a new ListDocumentTemplatesByModuleUseCase
func NewListDocumentTemplatesByModuleUseCase(
	repositories ListDocumentTemplatesByModuleRepositories,
	services ListDocumentTemplatesByModuleServices,
) *ListDocumentTemplatesByModuleUseCase {
	return &ListDocumentTemplatesByModuleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute lists document templates belonging to a specific module (e.g. "revenue")
func (uc *ListDocumentTemplatesByModuleUseCase) Execute(ctx context.Context, moduleKey string) (*documenttemplatepb.ListDocumentTemplatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDocumentTemplate, ports.ActionList); err != nil {
		return nil, err
	}

	if moduleKey == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "document_template.validation.module_key_required", "Module key is required [DEFAULT]"))
	}

	return uc.repositories.DocumentTemplate.ListDocumentTemplates(ctx, &documenttemplatepb.ListDocumentTemplatesRequest{
		Filters: &commonpb.FilterRequest{
			Logic: commonpb.FilterLogic_AND,
			Filters: []*commonpb.TypedFilter{
				{
					Field: "module_key",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Operator: commonpb.StringOperator_STRING_EQUALS,
							Value:    moduleKey,
						},
					},
				},
			},
		},
	})
}
