package outcome_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type ListByGroupRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type ListByGroupServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListByGroupUseCase handles the business logic for listing outcome criteria by group
type ListByGroupUseCase struct {
	repositories ListByGroupRepositories
	services     ListByGroupServices
}

// NewListByGroupUseCase creates a new ListByGroupUseCase
func NewListByGroupUseCase(
	repositories ListByGroupRepositories,
	services ListByGroupServices,
) *ListByGroupUseCase {
	return &ListByGroupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by group operation
func (uc *ListByGroupUseCase) Execute(ctx context.Context, req *pb.ListOutcomeCriteriasByGroupRequest) (*pb.ListOutcomeCriteriasByGroupResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.OutcomeCriteria, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list by group within a transaction
func (uc *ListByGroupUseCase) executeWithTransaction(ctx context.Context, req *pb.ListOutcomeCriteriasByGroupRequest) (*pb.ListOutcomeCriteriasByGroupResponse, error) {
	var result *pb.ListOutcomeCriteriasByGroupResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "outcome_criteria.errors.list_by_group_failed", "outcome criteria listing by group failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing outcome criteria by group
func (uc *ListByGroupUseCase) executeCore(ctx context.Context, req *pb.ListOutcomeCriteriasByGroupRequest) (*pb.ListOutcomeCriteriasByGroupResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.ListByGroup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.list_by_group_failed", "failed to list outcome criteria by group: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByGroupUseCase) validateInput(ctx context.Context, req *pb.ListOutcomeCriteriasByGroupRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.request_required", "request is required"))
	}
	if req.CriteriaGroupId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.group_id_required", "group ID is required"))
	}

	return nil
}
