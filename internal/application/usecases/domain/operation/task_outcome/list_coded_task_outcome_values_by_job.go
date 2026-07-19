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

// codedHistoricalReader is the OPTIONAL adapter capability that admits inactive
// (past-academic-year) instance+template ancestry. The postgres task_outcome
// adapter implements it; other providers (mock/firestore) do not, so the type
// assertion in ExecuteHistorical fails closed for them (empty result, never a
// live-path fallback that would silently misreport a past card as empty).
type codedHistoricalReader interface {
	ListCodedTaskOutcomeValuesByJobHistorical(context.Context, *pb.ListCodedTaskOutcomeValuesByJobRequest) (*pb.ListCodedTaskOutcomeValuesByJobResponse, error)
}

// ExecuteHistorical is the past-academic-year sibling of Execute. It applies the
// SAME task_outcome:list action gate (tenant/authorization unchanged), then
// delegates to the adapter's historical reader when available. When the bound
// adapter does not implement codedHistoricalReader the call fails closed with an
// empty, successful response rather than falling back to the active-only live path
// (which would silently return nothing for a past card).
func (uc *ListCodedTaskOutcomeValuesByJobUseCase) ExecuteHistorical(ctx context.Context, req *pb.ListCodedTaskOutcomeValuesByJobRequest) (*pb.ListCodedTaskOutcomeValuesByJobResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcome,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		req = &pb.ListCodedTaskOutcomeValuesByJobRequest{}
	}
	hist, ok := uc.repositories.TaskOutcome.(codedHistoricalReader)
	if !ok {
		return &pb.ListCodedTaskOutcomeValuesByJobResponse{Success: true}, nil
	}
	return hist.ListCodedTaskOutcomeValuesByJobHistorical(ctx, req)
}
