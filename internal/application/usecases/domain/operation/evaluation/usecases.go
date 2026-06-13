package evaluation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationresponsepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_response"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
)

// UseCases contains all evaluation-related use cases.
type UseCases struct {
	CreateEvaluation          *CreateEvaluationUseCase
	ReadEvaluation            *ReadEvaluationUseCase
	UpdateEvaluation          *UpdateEvaluationUseCase
	DeleteEvaluation          *DeleteEvaluationUseCase
	ListEvaluations           *ListEvaluationsUseCase
	GetEvaluationListPageData *GetEvaluationListPageDataUseCase
	GetEvaluationItemPageData *GetEvaluationItemPageDataUseCase
	SubmitEvaluation          *SubmitEvaluationUseCase
	SignOffEvaluation         *SignOffEvaluationUseCase
}

// EvaluationRepositories groups all repository dependencies for evaluation use cases.
type EvaluationRepositories struct {
	Evaluation         evaluationpb.EvaluationDomainServiceServer
	EvaluationResponse evaluationresponsepb.EvaluationResponseDomainServiceServer
	OutcomeCriteria    outcomecriteriapb.OutcomeCriteriaDomainServiceServer   // Submit snapshot source
	SubscriptionSeat   subscriptionseatpb.SubscriptionSeatDomainServiceServer // anchor-ownership validation
}

// EvaluationServices groups all business service dependencies.
type EvaluationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases wires all evaluation use cases.
func NewUseCases(repositories EvaluationRepositories, services EvaluationServices) *UseCases {
	return &UseCases{
		CreateEvaluation: NewCreateEvaluationUseCase(
			CreateEvaluationRepositories{Evaluation: repositories.Evaluation, SubscriptionSeat: repositories.SubscriptionSeat},
			CreateEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator, IDGenerator: services.IDGenerator},
		),
		ReadEvaluation: NewReadEvaluationUseCase(
			ReadEvaluationRepositories{Evaluation: repositories.Evaluation},
			ReadEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		UpdateEvaluation: NewUpdateEvaluationUseCase(
			UpdateEvaluationRepositories{Evaluation: repositories.Evaluation},
			UpdateEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		DeleteEvaluation: NewDeleteEvaluationUseCase(
			DeleteEvaluationRepositories{Evaluation: repositories.Evaluation},
			DeleteEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		ListEvaluations: NewListEvaluationsUseCase(
			ListEvaluationsRepositories{Evaluation: repositories.Evaluation},
			ListEvaluationsServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		GetEvaluationListPageData: NewGetEvaluationListPageDataUseCase(
			ListEvaluationsRepositories{Evaluation: repositories.Evaluation},
			ListEvaluationsServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		GetEvaluationItemPageData: NewGetEvaluationItemPageDataUseCase(
			ListEvaluationsRepositories{Evaluation: repositories.Evaluation},
			ListEvaluationsServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		SubmitEvaluation: NewSubmitEvaluationUseCase(
			SubmitEvaluationRepositories{Evaluation: repositories.Evaluation, EvaluationResponse: repositories.EvaluationResponse, OutcomeCriteria: repositories.OutcomeCriteria},
			SubmitEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
		SignOffEvaluation: NewSignOffEvaluationUseCase(
			SignOffEvaluationRepositories{Evaluation: repositories.Evaluation},
			SignOffEvaluationServices{Authorizer: services.Authorizer, Transactor: services.Transactor, Translator: services.Translator},
		),
	}
}
