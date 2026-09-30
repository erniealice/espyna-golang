package recovery_document_line

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// ListRecoveryDocumentLinesRepositories groups repository dependencies.
type ListRecoveryDocumentLinesRepositories struct {
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
}

// ListRecoveryDocumentLinesServices groups service dependencies.
type ListRecoveryDocumentLinesServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListRecoveryDocumentLinesUseCase lists rows of the caller's workspace (pagination honoured by the repository).
type ListRecoveryDocumentLinesUseCase struct {
	repositories ListRecoveryDocumentLinesRepositories
	services     ListRecoveryDocumentLinesServices
}

// NewListRecoveryDocumentLinesUseCase creates the use case with grouped dependencies.
func NewListRecoveryDocumentLinesUseCase(r ListRecoveryDocumentLinesRepositories, s ListRecoveryDocumentLinesServices) *ListRecoveryDocumentLinesUseCase {
	return &ListRecoveryDocumentLinesUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *ListRecoveryDocumentLinesUseCase) Execute(ctx context.Context, req *recoverydocumentlinepb.ListRecoveryDocumentLinesRequest) (*recoverydocumentlinepb.ListRecoveryDocumentLinesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RecoveryDocument, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if uc.repositories.RecoveryDocumentLine == nil {
		return nil, fmt.Errorf("recovery_document_line: repository unavailable")
	}
	if req == nil {
		req = &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{}
	}
	return uc.repositories.RecoveryDocumentLine.ListRecoveryDocumentLines(ctx, req)
}
