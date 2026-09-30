// Package document_series holds the use cases of document_series (domain revenue;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.3 "CreateDocumentSeries /
// update / list / read": code unique, retire only). Authorization: document_series:{list,read,
// create,update}. Number allocation (next_number) is done by IssueRecoveryDocuments under
// DocumentSeriesLocker.LockDocumentSeriesForUpdate; nothing here writes next_number after create.
// Every Execute takes and returns esqyma proto messages (C1); refusals are *Error with ErrorCode() (C2).
package document_series

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

func repoUnavailable() error { return fmt.Errorf("document_series: repository unavailable") }

// Repositories groups the repository dependencies (aggregate input).
type Repositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// Services groups the shared service dependencies (aggregate input).
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates the document_series use cases.
type UseCases struct {
	CreateDocumentSeries          *CreateDocumentSeriesUseCase
	UpdateDocumentSeries          *UpdateDocumentSeriesUseCase
	ReadDocumentSeries            *ReadDocumentSeriesUseCase
	ListDocumentSeries            *ListDocumentSeriesUseCase
	GetDocumentSeriesListPageData *GetDocumentSeriesListPageDataUseCase
}

// NewUseCases wires the document_series use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateDocumentSeries: NewCreateDocumentSeriesUseCase(
			CreateDocumentSeriesRepositories{DocumentSeries: r.DocumentSeries},
			CreateDocumentSeriesServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, IDGenerator: s.IDGenerator}),
		UpdateDocumentSeries: NewUpdateDocumentSeriesUseCase(
			UpdateDocumentSeriesRepositories{DocumentSeries: r.DocumentSeries},
			UpdateDocumentSeriesServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, Transactor: s.Transactor}),
		ReadDocumentSeries: NewReadDocumentSeriesUseCase(
			ReadDocumentSeriesRepositories{DocumentSeries: r.DocumentSeries},
			ReadDocumentSeriesServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ListDocumentSeries: NewListDocumentSeriesUseCase(
			ListDocumentSeriesRepositories{DocumentSeries: r.DocumentSeries},
			ListDocumentSeriesServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetDocumentSeriesListPageData: NewGetDocumentSeriesListPageDataUseCase(
			GetDocumentSeriesListPageDataRepositories{DocumentSeries: r.DocumentSeries},
			GetDocumentSeriesListPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
	}
}
