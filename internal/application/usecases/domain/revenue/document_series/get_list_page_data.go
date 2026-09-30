package document_series

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// GetDocumentSeriesListPageDataRepositories groups repository dependencies.
type GetDocumentSeriesListPageDataRepositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// GetDocumentSeriesListPageDataServices groups service dependencies.
type GetDocumentSeriesListPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetDocumentSeriesListPageDataUseCase returns a page of series with pagination (C11).
type GetDocumentSeriesListPageDataUseCase struct {
	repositories GetDocumentSeriesListPageDataRepositories
	services     GetDocumentSeriesListPageDataServices
}

// NewGetDocumentSeriesListPageDataUseCase creates the use case with grouped dependencies.
func NewGetDocumentSeriesListPageDataUseCase(r GetDocumentSeriesListPageDataRepositories, s GetDocumentSeriesListPageDataServices) *GetDocumentSeriesListPageDataUseCase {
	return &GetDocumentSeriesListPageDataUseCase{repositories: r, services: s}
}

// Execute returns one list page.
func (uc *GetDocumentSeriesListPageDataUseCase) Execute(ctx context.Context, req *pb.GetDocumentSeriesListPageDataRequest) (*pb.GetDocumentSeriesListPageDataResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repositories.DocumentSeries == nil {
		return nil, repoUnavailable()
	}
	if req == nil {
		req = &pb.GetDocumentSeriesListPageDataRequest{}
	}
	resp, err := uc.repositories.DocumentSeries.GetDocumentSeriesListPageData(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("document_series", "list page document_series", err)
	}
	return resp, nil
}
