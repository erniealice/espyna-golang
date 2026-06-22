package scoring_component

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
)

// UseCases aggregates the scoring_component CRUD + page-data use cases.
type UseCases struct {
	CreateScoringComponent          *CreateScoringComponentUseCase
	ReadScoringComponent            *ReadScoringComponentUseCase
	UpdateScoringComponent          *UpdateScoringComponentUseCase
	DeleteScoringComponent          *DeleteScoringComponentUseCase
	ListScoringComponents           *ListScoringComponentsUseCase
	GetScoringComponentListPageData *GetScoringComponentListPageDataUseCase
	GetScoringComponentItemPageData *GetScoringComponentItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ScoringComponent pb.ScoringComponentDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the scoring_component use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ScoringComponent
	return &UseCases{
		CreateScoringComponent:          NewCreateScoringComponentUseCase(CreateScoringComponentRepositories{ScoringComponent: repo}, CreateScoringComponentServices(s)),
		ReadScoringComponent:            NewReadScoringComponentUseCase(ReadScoringComponentRepositories{ScoringComponent: repo}, ReadScoringComponentServices(s)),
		UpdateScoringComponent:          NewUpdateScoringComponentUseCase(UpdateScoringComponentRepositories{ScoringComponent: repo}, UpdateScoringComponentServices(s)),
		DeleteScoringComponent:          NewDeleteScoringComponentUseCase(DeleteScoringComponentRepositories{ScoringComponent: repo}, DeleteScoringComponentServices(s)),
		ListScoringComponents:           NewListScoringComponentsUseCase(ListScoringComponentsRepositories{ScoringComponent: repo}, ListScoringComponentsServices(s)),
		GetScoringComponentListPageData: NewGetScoringComponentListPageDataUseCase(GetScoringComponentListPageDataRepositories{ScoringComponent: repo}, GetScoringComponentListPageDataServices(s)),
		GetScoringComponentItemPageData: NewGetScoringComponentItemPageDataUseCase(GetScoringComponentItemPageDataRepositories{ScoringComponent: repo}, GetScoringComponentItemPageDataServices(s)),
	}
}
