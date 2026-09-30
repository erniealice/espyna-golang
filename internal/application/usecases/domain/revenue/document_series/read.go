package document_series

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// ReadDocumentSeriesRepositories groups repository dependencies.
type ReadDocumentSeriesRepositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// ReadDocumentSeriesServices groups service dependencies.
type ReadDocumentSeriesServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadDocumentSeriesUseCase reads one series (foreign or missing id = not_found).
type ReadDocumentSeriesUseCase struct {
	repositories ReadDocumentSeriesRepositories
	services     ReadDocumentSeriesServices
}

// NewReadDocumentSeriesUseCase creates the use case with grouped dependencies.
func NewReadDocumentSeriesUseCase(r ReadDocumentSeriesRepositories, s ReadDocumentSeriesServices) *ReadDocumentSeriesUseCase {
	return &ReadDocumentSeriesUseCase{repositories: r, services: s}
}

// Execute reads the series.
func (uc *ReadDocumentSeriesUseCase) Execute(ctx context.Context, req *pb.ReadDocumentSeriesRequest) (*pb.ReadDocumentSeriesResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "document_series", invalid("id is required"))
	}
	row, err := readSeries(ctx, uc.repositories.DocumentSeries, req.Data.GetId())
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "document_series", err)
	}
	return &pb.ReadDocumentSeriesResponse{Data: []*pb.DocumentSeries{row}, Success: true}, nil
}
