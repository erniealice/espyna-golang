package scoring_component_criteria

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
)

// UseCases aggregates the scoring_component_criteria CRUD + page-data use cases.
type UseCases struct {
	CreateScoringComponentCriteria          *CreateScoringComponentCriteriaUseCase
	ReadScoringComponentCriteria            *ReadScoringComponentCriteriaUseCase
	UpdateScoringComponentCriteria          *UpdateScoringComponentCriteriaUseCase
	DeleteScoringComponentCriteria          *DeleteScoringComponentCriteriaUseCase
	ListScoringComponentCriterias           *ListScoringComponentCriteriasUseCase
	GetScoringComponentCriteriaListPageData *GetScoringComponentCriteriaListPageDataUseCase
	GetScoringComponentCriteriaItemPageData *GetScoringComponentCriteriaItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ScoringComponentCriteria pb.ScoringComponentCriteriaDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the scoring_component_criteria use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ScoringComponentCriteria
	return &UseCases{
		CreateScoringComponentCriteria:          NewCreateScoringComponentCriteriaUseCase(CreateScoringComponentCriteriaRepositories{ScoringComponentCriteria: repo}, CreateScoringComponentCriteriaServices(s)),
		ReadScoringComponentCriteria:            NewReadScoringComponentCriteriaUseCase(ReadScoringComponentCriteriaRepositories{ScoringComponentCriteria: repo}, ReadScoringComponentCriteriaServices(s)),
		UpdateScoringComponentCriteria:          NewUpdateScoringComponentCriteriaUseCase(UpdateScoringComponentCriteriaRepositories{ScoringComponentCriteria: repo}, UpdateScoringComponentCriteriaServices(s)),
		DeleteScoringComponentCriteria:          NewDeleteScoringComponentCriteriaUseCase(DeleteScoringComponentCriteriaRepositories{ScoringComponentCriteria: repo}, DeleteScoringComponentCriteriaServices(s)),
		ListScoringComponentCriterias:           NewListScoringComponentCriteriasUseCase(ListScoringComponentCriteriasRepositories{ScoringComponentCriteria: repo}, ListScoringComponentCriteriasServices(s)),
		GetScoringComponentCriteriaListPageData: NewGetScoringComponentCriteriaListPageDataUseCase(GetScoringComponentCriteriaListPageDataRepositories{ScoringComponentCriteria: repo}, GetScoringComponentCriteriaListPageDataServices(s)),
		GetScoringComponentCriteriaItemPageData: NewGetScoringComponentCriteriaItemPageDataUseCase(GetScoringComponentCriteriaItemPageDataRepositories{ScoringComponentCriteria: repo}, GetScoringComponentCriteriaItemPageDataServices(s)),
	}
}
