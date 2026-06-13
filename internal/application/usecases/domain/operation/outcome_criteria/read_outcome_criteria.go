package outcome_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type ReadOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type ReadOutcomeCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadOutcomeCriteriaUseCase handles the business logic for reading outcome criteria
type ReadOutcomeCriteriaUseCase struct {
	repositories ReadOutcomeCriteriaRepositories
	services     ReadOutcomeCriteriaServices
}

// NewReadOutcomeCriteriaUseCase creates a new ReadOutcomeCriteriaUseCase
func NewReadOutcomeCriteriaUseCase(
	repositories ReadOutcomeCriteriaRepositories,
	services ReadOutcomeCriteriaServices,
) *ReadOutcomeCriteriaUseCase {
	return &ReadOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read outcome criteria operation
func (uc *ReadOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.OutcomeCriteria,
		Action: entityid.ActionRead,
	}); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	var result *pb.ReadOutcomeCriteriaResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for reading an outcome criteria
func (uc *ReadOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) (*pb.ReadOutcomeCriteriaResponse, error) {
	resp, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.not_found", "[ERR-DEFAULT] Outcome criteria not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.not_found", "[ERR-DEFAULT] Outcome criteria not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadOutcomeCriteriaUseCase) validateInput(ctx context.Context, req *pb.ReadOutcomeCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	return nil
}
