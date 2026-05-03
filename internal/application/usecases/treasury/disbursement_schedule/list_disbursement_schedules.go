package disbursementschedule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
)

const entityDisbursementSchedule = "disbursement_schedule"

// ListDisbursementSchedulesRepositories groups repository dependencies for the use case.
type ListDisbursementSchedulesRepositories struct {
	DisbursementSchedule disbursementschedulepb.DisbursementScheduleDomainServiceServer
}

// ListDisbursementSchedulesServices groups service dependencies for the use case.
type ListDisbursementSchedulesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListDisbursementSchedulesUseCase lists disbursement schedules.
type ListDisbursementSchedulesUseCase struct {
	repositories ListDisbursementSchedulesRepositories
	services     ListDisbursementSchedulesServices
}

// NewListDisbursementSchedulesUseCase creates a new ListDisbursementSchedulesUseCase.
func NewListDisbursementSchedulesUseCase(
	repositories ListDisbursementSchedulesRepositories,
	services ListDisbursementSchedulesServices,
) *ListDisbursementSchedulesUseCase {
	return &ListDisbursementSchedulesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list disbursement schedules operation.
func (uc *ListDisbursementSchedulesUseCase) Execute(ctx context.Context, req *disbursementschedulepb.ListDisbursementSchedulesRequest) (*disbursementschedulepb.ListDisbursementSchedulesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDisbursementSchedule, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "disbursement_schedule.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.DisbursementSchedule.ListDisbursementSchedules(ctx, req)
}
