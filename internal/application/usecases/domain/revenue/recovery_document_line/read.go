package recovery_document_line

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// ReadRecoveryDocumentLineRepositories groups repository dependencies.
type ReadRecoveryDocumentLineRepositories struct {
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
}

// ReadRecoveryDocumentLineServices groups service dependencies.
type ReadRecoveryDocumentLineServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadRecoveryDocumentLineUseCase reads one row by id (a foreign-workspace and a missing id are both "not found").
type ReadRecoveryDocumentLineUseCase struct {
	repositories ReadRecoveryDocumentLineRepositories
	services     ReadRecoveryDocumentLineServices
}

// NewReadRecoveryDocumentLineUseCase creates the use case with grouped dependencies.
func NewReadRecoveryDocumentLineUseCase(r ReadRecoveryDocumentLineRepositories, s ReadRecoveryDocumentLineServices) *ReadRecoveryDocumentLineUseCase {
	return &ReadRecoveryDocumentLineUseCase{repositories: r, services: s}
}

// Execute runs the use case.
func (uc *ReadRecoveryDocumentLineUseCase) Execute(ctx context.Context, req *recoverydocumentlinepb.ReadRecoveryDocumentLineRequest) (*recoverydocumentlinepb.ReadRecoveryDocumentLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RecoveryDocument, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if uc.repositories.RecoveryDocumentLine == nil {
		return nil, fmt.Errorf("recovery_document_line: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("recovery_document_line: id is required")
	}
	return uc.repositories.RecoveryDocumentLine.ReadRecoveryDocumentLine(ctx, req)
}
