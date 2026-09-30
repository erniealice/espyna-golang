package document_series

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// ListDocumentSeriesRepositories groups repository dependencies.
type ListDocumentSeriesRepositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// ListDocumentSeriesServices groups service dependencies.
type ListDocumentSeriesServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListDocumentSeriesUseCase lists series of the caller's workspace (pagination honoured by the repository).
type ListDocumentSeriesUseCase struct {
	repositories ListDocumentSeriesRepositories
	services     ListDocumentSeriesServices
}

// NewListDocumentSeriesUseCase creates the use case with grouped dependencies.
func NewListDocumentSeriesUseCase(r ListDocumentSeriesRepositories, s ListDocumentSeriesServices) *ListDocumentSeriesUseCase {
	return &ListDocumentSeriesUseCase{repositories: r, services: s}
}

// Execute lists series.
func (uc *ListDocumentSeriesUseCase) Execute(ctx context.Context, req *pb.ListDocumentSeriesRequest) (*pb.ListDocumentSeriesResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repositories.DocumentSeries == nil {
		return nil, repoUnavailable()
	}
	if req == nil {
		req = &pb.ListDocumentSeriesRequest{}
	}
	resp, err := uc.repositories.DocumentSeries.ListDocumentSeries(ctx, req)
	if err != nil {
		return nil, usecaseerr.RepoErr("document_series", "list document_series", err)
	}
	return resp, nil
}
