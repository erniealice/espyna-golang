package score_scale_band

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// UseCases aggregates the score_scale_band CRUD + page-data use cases.
type UseCases struct {
	CreateScoreScaleBand          *CreateScoreScaleBandUseCase
	ReadScoreScaleBand            *ReadScoreScaleBandUseCase
	UpdateScoreScaleBand          *UpdateScoreScaleBandUseCase
	DeleteScoreScaleBand          *DeleteScoreScaleBandUseCase
	ListScoreScaleBands           *ListScoreScaleBandsUseCase
	GetScoreScaleBandListPageData *GetScoreScaleBandListPageDataUseCase
	GetScoreScaleBandItemPageData *GetScoreScaleBandItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the score_scale_band use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ScoreScaleBand
	return &UseCases{
		CreateScoreScaleBand:          NewCreateScoreScaleBandUseCase(CreateScoreScaleBandRepositories{ScoreScaleBand: repo}, CreateScoreScaleBandServices(s)),
		ReadScoreScaleBand:            NewReadScoreScaleBandUseCase(ReadScoreScaleBandRepositories{ScoreScaleBand: repo}, ReadScoreScaleBandServices(s)),
		UpdateScoreScaleBand:          NewUpdateScoreScaleBandUseCase(UpdateScoreScaleBandRepositories{ScoreScaleBand: repo}, UpdateScoreScaleBandServices(s)),
		DeleteScoreScaleBand:          NewDeleteScoreScaleBandUseCase(DeleteScoreScaleBandRepositories{ScoreScaleBand: repo}, DeleteScoreScaleBandServices(s)),
		ListScoreScaleBands:           NewListScoreScaleBandsUseCase(ListScoreScaleBandsRepositories{ScoreScaleBand: repo}, ListScoreScaleBandsServices(s)),
		GetScoreScaleBandListPageData: NewGetScoreScaleBandListPageDataUseCase(GetScoreScaleBandListPageDataRepositories{ScoreScaleBand: repo}, GetScoreScaleBandListPageDataServices(s)),
		GetScoreScaleBandItemPageData: NewGetScoreScaleBandItemPageDataUseCase(GetScoreScaleBandItemPageDataRepositories{ScoreScaleBand: repo}, GetScoreScaleBandItemPageDataServices(s)),
	}
}
