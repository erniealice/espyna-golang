package scoring_scheme

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

// UseCases aggregates the scoring_scheme CRUD + page-data use cases.
type UseCases struct {
	CreateScoringScheme          *CreateScoringSchemeUseCase
	ReadScoringScheme            *ReadScoringSchemeUseCase
	UpdateScoringScheme          *UpdateScoringSchemeUseCase
	DeleteScoringScheme          *DeleteScoringSchemeUseCase
	ListScoringSchemes           *ListScoringSchemesUseCase
	GetScoringSchemeListPageData *GetScoringSchemeListPageDataUseCase
	GetScoringSchemeItemPageData *GetScoringSchemeItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ScoringScheme pb.ScoringSchemeDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the scoring_scheme use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ScoringScheme
	return &UseCases{
		CreateScoringScheme:          NewCreateScoringSchemeUseCase(CreateScoringSchemeRepositories{ScoringScheme: repo}, CreateScoringSchemeServices(s)),
		ReadScoringScheme:            NewReadScoringSchemeUseCase(ReadScoringSchemeRepositories{ScoringScheme: repo}, ReadScoringSchemeServices(s)),
		UpdateScoringScheme:          NewUpdateScoringSchemeUseCase(UpdateScoringSchemeRepositories{ScoringScheme: repo}, UpdateScoringSchemeServices(s)),
		DeleteScoringScheme:          NewDeleteScoringSchemeUseCase(DeleteScoringSchemeRepositories{ScoringScheme: repo}, DeleteScoringSchemeServices(s)),
		ListScoringSchemes:           NewListScoringSchemesUseCase(ListScoringSchemesRepositories{ScoringScheme: repo}, ListScoringSchemesServices(s)),
		GetScoringSchemeListPageData: NewGetScoringSchemeListPageDataUseCase(GetScoringSchemeListPageDataRepositories{ScoringScheme: repo}, GetScoringSchemeListPageDataServices(s)),
		GetScoringSchemeItemPageData: NewGetScoringSchemeItemPageDataUseCase(GetScoringSchemeItemPageDataRepositories{ScoringScheme: repo}, GetScoringSchemeItemPageDataServices(s)),
	}
}
