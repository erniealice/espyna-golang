package disbursementschedule

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
)

// DisbursementScheduleRepositories groups all repository dependencies for disbursement schedule use cases.
type DisbursementScheduleRepositories struct {
	DisbursementSchedule disbursementschedulepb.DisbursementScheduleDomainServiceServer
}

// DisbursementScheduleServices groups all business service dependencies for disbursement schedule use cases.
type DisbursementScheduleServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all disbursement-schedule-related use cases.
type UseCases struct {
	ListDisbursementSchedules *ListDisbursementSchedulesUseCase
}

// NewUseCases creates a new collection of disbursement schedule use cases.
func NewUseCases(
	repositories DisbursementScheduleRepositories,
	services DisbursementScheduleServices,
) *UseCases {
	listRepos := ListDisbursementSchedulesRepositories(repositories)
	listSvcs := ListDisbursementSchedulesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		ListDisbursementSchedules: NewListDisbursementSchedulesUseCase(listRepos, listSvcs),
	}
}
