package recovery_document

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
)

// ListRecoveryDocumentsRepositories groups repository dependencies.
type ListRecoveryDocumentsRepositories struct {
	RecoveryDocument recoverydocumentpb.RecoveryDocumentDomainServiceServer
}

// ListRecoveryDocumentsServices groups service dependencies.
type ListRecoveryDocumentsServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListRecoveryDocumentsUseCase lists documents of the caller's workspace, optionally by status
// (request.status; UNSPECIFIED = all). Pagination is honoured by the repository. Gate
// recovery_document:list.
type ListRecoveryDocumentsUseCase struct {
	repositories ListRecoveryDocumentsRepositories
	services     ListRecoveryDocumentsServices
}

// NewListRecoveryDocumentsUseCase creates the use case with grouped dependencies.
func NewListRecoveryDocumentsUseCase(r ListRecoveryDocumentsRepositories, s ListRecoveryDocumentsServices) *ListRecoveryDocumentsUseCase {
	return &ListRecoveryDocumentsUseCase{repositories: r, services: s}
}

// Execute lists documents.
func (uc *ListRecoveryDocumentsUseCase) Execute(ctx context.Context, req *recoverydocumentpb.ListRecoveryDocumentsRequest) (*recoverydocumentpb.ListRecoveryDocumentsResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repositories.RecoveryDocument == nil {
		return nil, fmt.Errorf("recovery_document: repository unavailable")
	}
	if req == nil {
		req = &recoverydocumentpb.ListRecoveryDocumentsRequest{}
	}
	if st := req.GetStatus(); st != recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_UNSPECIFIED {
		filters := &commonpb.FilterRequest{}
		if req.Filters != nil {
			filters.Logic = req.Filters.Logic
			filters.Filters = append(filters.Filters, req.Filters.Filters...)
		}
		filters.Filters = append(filters.Filters, stringEq("status", st.String()))
		req = &recoverydocumentpb.ListRecoveryDocumentsRequest{Search: req.Search, Filters: filters, Sort: req.Sort, Pagination: req.Pagination, Status: req.Status}
	}
	resp, err := uc.repositories.RecoveryDocument.ListRecoveryDocuments(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list recovery_documents", err)
	}
	return resp, nil
}
