package score_scale

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// UseCases aggregates the score_scale CRUD + page-data use cases.
type UseCases struct {
	CreateScoreScale          *CreateScoreScaleUseCase
	ReadScoreScale            *ReadScoreScaleUseCase
	UpdateScoreScale          *UpdateScoreScaleUseCase
	DeleteScoreScale          *DeleteScoreScaleUseCase
	ListScoreScales           *ListScoreScalesUseCase
	GetScoreScaleListPageData *GetScoreScaleListPageDataUseCase
	GetScoreScaleItemPageData *GetScoreScaleItemPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	ScoreScale pb.ScoreScaleDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the score_scale use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.ScoreScale
	return &UseCases{
		CreateScoreScale:          NewCreateScoreScaleUseCase(CreateScoreScaleRepositories{ScoreScale: repo}, CreateScoreScaleServices(s)),
		ReadScoreScale:            NewReadScoreScaleUseCase(ReadScoreScaleRepositories{ScoreScale: repo}, ReadScoreScaleServices(s)),
		UpdateScoreScale:          NewUpdateScoreScaleUseCase(UpdateScoreScaleRepositories{ScoreScale: repo}, UpdateScoreScaleServices(s)),
		DeleteScoreScale:          NewDeleteScoreScaleUseCase(DeleteScoreScaleRepositories{ScoreScale: repo}, DeleteScoreScaleServices(s)),
		ListScoreScales:           NewListScoreScalesUseCase(ListScoreScalesRepositories{ScoreScale: repo}, ListScoreScalesServices(s)),
		GetScoreScaleListPageData: NewGetScoreScaleListPageDataUseCase(GetScoreScaleListPageDataRepositories{ScoreScale: repo}, GetScoreScaleListPageDataServices(s)),
		GetScoreScaleItemPageData: NewGetScoreScaleItemPageDataUseCase(GetScoreScaleItemPageDataRepositories{ScoreScale: repo}, GetScoreScaleItemPageDataServices(s)),
	}
}
