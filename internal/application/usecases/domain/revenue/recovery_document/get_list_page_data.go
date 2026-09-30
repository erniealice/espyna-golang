package recovery_document

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
)

// GetRecoveryDocumentListPageDataRepositories groups repository dependencies.
type GetRecoveryDocumentListPageDataRepositories struct {
	RecoveryDocument recoverydocumentpb.RecoveryDocumentDomainServiceServer
}

// GetRecoveryDocumentListPageDataServices groups service dependencies.
type GetRecoveryDocumentListPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRecoveryDocumentListPageDataUseCase returns a page of rows with pagination (C11).
type GetRecoveryDocumentListPageDataUseCase struct {
	repositories GetRecoveryDocumentListPageDataRepositories
	services     GetRecoveryDocumentListPageDataServices
}

// NewGetRecoveryDocumentListPageDataUseCase creates the use case with grouped dependencies.
func NewGetRecoveryDocumentListPageDataUseCase(r GetRecoveryDocumentListPageDataRepositories, s GetRecoveryDocumentListPageDataServices) *GetRecoveryDocumentListPageDataUseCase {
	return &GetRecoveryDocumentListPageDataUseCase{repositories: r, services: s}
}

// Execute returns one list page.
func (uc *GetRecoveryDocumentListPageDataUseCase) Execute(ctx context.Context, req *recoverydocumentpb.GetRecoveryDocumentListPageDataRequest) (*recoverydocumentpb.GetRecoveryDocumentListPageDataResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repositories.RecoveryDocument == nil {
		return nil, fmt.Errorf("recovery_document: repository unavailable")
	}
	if req == nil {
		req = &recoverydocumentpb.GetRecoveryDocumentListPageDataRequest{}
	}
	resp, err := uc.repositories.RecoveryDocument.GetRecoveryDocumentListPageData(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list page recovery_documents", err)
	}
	return resp, nil
}
