package evaluation

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

// ListEvaluationsRepositories groups all repository dependencies.
type ListEvaluationsRepositories struct {
	Evaluation evaluationpb.EvaluationDomainServiceServer
}

// ListEvaluationsServices groups all business service dependencies.
type ListEvaluationsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListEvaluationsUseCase lists evaluations. The IDOR workspace_id + client_id +
// visibility_type scoping is enforced in the adapter's GetEvaluationListPageData
// query predicate (keyed off the session's workspace + acting_as_client_id), so
// the client path is fail-closed at the query: a client caller only ever sees
// its own non-internal rows. This use case applies the authcheck gate.
type ListEvaluationsUseCase struct {
	repositories ListEvaluationsRepositories
	services     ListEvaluationsServices
}

func NewListEvaluationsUseCase(repositories ListEvaluationsRepositories, services ListEvaluationsServices) *ListEvaluationsUseCase {
	return &ListEvaluationsUseCase{repositories: repositories, services: services}
}

func (uc *ListEvaluationsUseCase) Execute(ctx context.Context, req *evaluationpb.ListEvaluationsRequest) (*evaluationpb.ListEvaluationsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.Evaluation.ListEvaluations(ctx, req)
}

// GetEvaluationListPageDataUseCase wraps the paginated list page data.
type GetEvaluationListPageDataUseCase struct {
	repositories ListEvaluationsRepositories
	services     ListEvaluationsServices
}

func NewGetEvaluationListPageDataUseCase(repositories ListEvaluationsRepositories, services ListEvaluationsServices) *GetEvaluationListPageDataUseCase {
	return &GetEvaluationListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetEvaluationListPageDataUseCase) Execute(ctx context.Context, req *evaluationpb.GetEvaluationListPageDataRequest) (*evaluationpb.GetEvaluationListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.Evaluation.GetEvaluationListPageData(ctx, req)
}

// GetEvaluationItemPageDataUseCase wraps the item page data (IDOR scoped in the adapter).
type GetEvaluationItemPageDataUseCase struct {
	repositories ListEvaluationsRepositories
	services     ListEvaluationsServices
}

func NewGetEvaluationItemPageDataUseCase(repositories ListEvaluationsRepositories, services ListEvaluationsServices) *GetEvaluationItemPageDataUseCase {
	return &GetEvaluationItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetEvaluationItemPageDataUseCase) Execute(ctx context.Context, req *evaluationpb.GetEvaluationItemPageDataRequest) (*evaluationpb.GetEvaluationItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Evaluation,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.Evaluation.GetEvaluationItemPageData(ctx, req)
}
