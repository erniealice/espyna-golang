// Package recovery_document_line holds the read use cases of the recovery_document_line entity (domain revenue;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.4). recovery_document_line rows are
// immutable financial records: this package never mutates them (writes happen in the behaviour
// use cases that own them) and the adapter refuses Delete (C6). Every Execute takes and returns
// esqyma proto messages (C1). Authorization: recovery_document:{list,read} (fail closed).
package recovery_document_line

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// Repositories groups the repository dependencies (aggregate input).
type Repositories struct {
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
}

// Services groups the shared service dependencies (aggregate input).
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates the recovery_document_line read use cases.
type UseCases struct {
	ListRecoveryDocumentLines           *ListRecoveryDocumentLinesUseCase
	GetRecoveryDocumentLineListPageData *GetRecoveryDocumentLineListPageDataUseCase
	ReadRecoveryDocumentLine            *ReadRecoveryDocumentLineUseCase
}

// NewUseCases wires the recovery_document_line read use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListRecoveryDocumentLines: NewListRecoveryDocumentLinesUseCase(
			ListRecoveryDocumentLinesRepositories{RecoveryDocumentLine: r.RecoveryDocumentLine},
			ListRecoveryDocumentLinesServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetRecoveryDocumentLineListPageData: NewGetRecoveryDocumentLineListPageDataUseCase(
			GetRecoveryDocumentLineListPageDataRepositories{RecoveryDocumentLine: r.RecoveryDocumentLine},
			GetRecoveryDocumentLineListPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ReadRecoveryDocumentLine: NewReadRecoveryDocumentLineUseCase(
			ReadRecoveryDocumentLineRepositories{RecoveryDocumentLine: r.RecoveryDocumentLine},
			ReadRecoveryDocumentLineServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
	}
}
