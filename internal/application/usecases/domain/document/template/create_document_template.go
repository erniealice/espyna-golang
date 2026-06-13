package template

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
)

const entityDocumentTemplate = "document_template"

// CreateDocumentTemplateRepositories groups all repository dependencies
type CreateDocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// CreateDocumentTemplateServices groups all business service dependencies
type CreateDocumentTemplateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateDocumentTemplateUseCase handles the business logic for creating document templates
type CreateDocumentTemplateUseCase struct {
	repositories CreateDocumentTemplateRepositories
	services     CreateDocumentTemplateServices
}

// NewCreateDocumentTemplateUseCase creates use case with grouped dependencies
func NewCreateDocumentTemplateUseCase(
	repositories CreateDocumentTemplateRepositories,
	services CreateDocumentTemplateServices,
) *CreateDocumentTemplateUseCase {
	return &CreateDocumentTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create document template operation
func (uc *CreateDocumentTemplateUseCase) Execute(ctx context.Context, req *documenttemplatepb.CreateDocumentTemplateRequest) (*documenttemplatepb.CreateDocumentTemplateResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityDocumentTemplate,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *documenttemplatepb.CreateDocumentTemplateResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("document template creation failed: %w", err)
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

func (uc *CreateDocumentTemplateUseCase) executeCore(ctx context.Context, req *documenttemplatepb.CreateDocumentTemplateRequest) (*documenttemplatepb.CreateDocumentTemplateResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "document_template.validation.data_required", "Document template data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.DocumentTemplate.CreateDocumentTemplate(ctx, req)
}
