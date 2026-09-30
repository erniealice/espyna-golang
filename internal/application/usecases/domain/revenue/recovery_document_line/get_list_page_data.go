package recovery_document_line

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// GetRecoveryDocumentLineListPageDataRepositories groups repository dependencies.
type GetRecoveryDocumentLineListPageDataRepositories struct {
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
}

// GetRecoveryDocumentLineListPageDataServices groups service dependencies.
type GetRecoveryDocumentLineListPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRecoveryDocumentLineListPageDataUseCase returns a page of rows with pagination (C11).
type GetRecoveryDocumentLineListPageDataUseCase struct {
	repositories GetRecoveryDocumentLineListPageDataRepositories
	services     GetRecoveryDocumentLineListPageDataServices
}

// NewGetRecoveryDocumentLineListPageDataUseCase creates the use case with grouped dependencies.
func NewGetRecoveryDocumentLineListPageDataUseCase(r GetRecoveryDocumentLineListPageDataRepositories, s GetRecoveryDocumentLineListPageDataServices) *GetRecoveryDocumentLineListPageDataUseCase {
	return &GetRecoveryDocumentLineListPageDataUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *GetRecoveryDocumentLineListPageDataUseCase) Execute(ctx context.Context, req *recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataRequest) (*recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RecoveryDocument, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if uc.repositories.RecoveryDocumentLine == nil {
		return nil, fmt.Errorf("recovery_document_line: repository unavailable")
	}
	if req == nil {
		req = &recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataRequest{}
	}
	return uc.repositories.RecoveryDocumentLine.GetRecoveryDocumentLineListPageData(ctx, req)
}
