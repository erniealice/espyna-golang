package task_outcome

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

type ListCodedTaskOutcomeValuesByJobRepositories struct {
	TaskOutcome pb.TaskOutcomeDomainServiceServer
}

type ListCodedTaskOutcomeValuesByJobServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListCodedTaskOutcomeValuesByJobUseCase gates task_outcome:list then delegates
// to the ownership-joined latest-cell repository read. Workspace isolation is
// enforced in the adapter from trusted context (never the request), and the job
// allowlist on the request is the caller's already-authorized set.
type ListCodedTaskOutcomeValuesByJobUseCase struct {
	repositories ListCodedTaskOutcomeValuesByJobRepositories
	services     ListCodedTaskOutcomeValuesByJobServices
}

// NewListCodedTaskOutcomeValuesByJobUseCase creates the use case.
func NewListCodedTaskOutcomeValuesByJobUseCase(
	repositories ListCodedTaskOutcomeValuesByJobRepositories,
	services ListCodedTaskOutcomeValuesByJobServices,
) *ListCodedTaskOutcomeValuesByJobUseCase {
	return &ListCodedTaskOutcomeValuesByJobUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute action-gates on task_outcome:list then delegates to the repository.
func (uc *ListCodedTaskOutcomeValuesByJobUseCase) Execute(ctx context.Context, req *pb.ListCodedTaskOutcomeValuesByJobRequest) (*pb.ListCodedTaskOutcomeValuesByJobResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		req = &pb.ListCodedTaskOutcomeValuesByJobRequest{}
	}
	return uc.repositories.TaskOutcome.ListCodedTaskOutcomeValuesByJob(ctx, req)
}
