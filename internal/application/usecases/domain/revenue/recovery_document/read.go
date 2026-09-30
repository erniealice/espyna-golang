package recovery_document

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// ReadRecoveryDocumentRepositories groups repository dependencies.
type ReadRecoveryDocumentRepositories struct {
	RecoveryDocument     recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
}

// ReadRecoveryDocumentServices groups service dependencies.
type ReadRecoveryDocumentServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadRecoveryDocumentUseCase reads one document with its lines and the credit notes issued against
// it (response.data[0], .lines, .credit_notes). A foreign or missing id is not_found. Gate
// recovery_document:read.
type ReadRecoveryDocumentUseCase struct {
	repositories ReadRecoveryDocumentRepositories
	services     ReadRecoveryDocumentServices
}

// NewReadRecoveryDocumentUseCase creates the use case with grouped dependencies.
func NewReadRecoveryDocumentUseCase(r ReadRecoveryDocumentRepositories, s ReadRecoveryDocumentServices) *ReadRecoveryDocumentUseCase {
	return &ReadRecoveryDocumentUseCase{repositories: r, services: s}
}

// Execute reads the document detail.
func (uc *ReadRecoveryDocumentUseCase) Execute(ctx context.Context, req *recoverydocumentpb.ReadRecoveryDocumentRequest) (*recoverydocumentpb.ReadRecoveryDocumentResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionRead); err != nil {
		return nil, err
	}
	r := uc.repositories
	if r.RecoveryDocument == nil || r.RecoveryDocumentLine == nil {
		return nil, fmt.Errorf("recovery_document: repositories unavailable")
	}
	if req == nil || req.Data == nil || blank(req.Data.Id) {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", errNotFound)
	}
	id := req.Data.Id
	resp, err := r.RecoveryDocument.ReadRecoveryDocument(ctx, req)
	if err != nil && !usecaseerr.IsNotFound(err) {
		return nil, usecaseerr.RepoErr("recovery_document", "read recovery_document", err, id)
	}
	if err != nil || resp == nil || len(resp.Data) == 0 {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", errNotFound)
	}
	lines, err := linesOf(ctx, r.RecoveryDocumentLine, id)
	if err != nil {
		return nil, err
	}
	creditNotes, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentpb.RecoveryDocument, error) {
		cr, err := r.RecoveryDocument.ListRecoveryDocuments(ctx, &recoverydocumentpb.ListRecoveryDocumentsRequest{Filters: listdata.EqFilter("corrects_document_id", id), Sort: s, Pagination: p})
		return cr.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list credit notes", err, id)
	}
	return &recoverydocumentpb.ReadRecoveryDocumentResponse{Data: resp.Data[:1], Lines: lines, CreditNotes: creditNotes, Success: true}, nil
}
