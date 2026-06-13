package disbursement

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// DisbursementRepositories groups all repository dependencies for disbursement use cases
type DisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// DisbursementServices groups all business service dependencies for disbursement use cases
type DisbursementServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all disbursement-related use cases
type UseCases struct {
	CreateDisbursement *CreateDisbursementUseCase
	ReadDisbursement   *ReadDisbursementUseCase
	UpdateDisbursement *UpdateDisbursementUseCase
	DeleteDisbursement *DeleteDisbursementUseCase
	ListDisbursements  *ListDisbursementsUseCase

	// 20260518-hexagonal-strict-adherence Phase 1.C — advance use cases (buying
	// side) folded back into the entity sub-aggregate from the prior
	// treasury_disbursement/ parallel home. Constructed by treasury.NewUseCases
	// after the CRUD use cases above are wired, because the advance flows
	// depend on UpdateDisbursement per Q1-B caller routing.
	AmortizeAdvance           *AmortizeAdvanceDisbursementUseCase
	SettleUnscheduledAdvance  *SettleUnscheduledAdvanceUseCase
	RefundUnscheduledAdvance  *RefundUnscheduledAdvanceUseCase
	CancelAdvance             *CancelAdvanceUseCase
	RecognizeMilestoneAdvance *RecognizeMilestoneAdvanceDisbursementUseCase
	ListAdvancesForDashboard  *ListAdvanceDisbursementsForDashboardUseCase
}

// NewUseCases creates a new collection of disbursement use cases
func NewUseCases(
	repositories DisbursementRepositories,
	services DisbursementServices,
) *UseCases {
	createRepos := CreateDisbursementRepositories(repositories)
	createServices := CreateDisbursementServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadDisbursementRepositories(repositories)
	readServices := ReadDisbursementServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateDisbursementRepositories(repositories)
	updateServices := UpdateDisbursementServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteDisbursementRepositories(repositories)
	deleteServices := DeleteDisbursementServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListDisbursementsRepositories(repositories)
	listServices := ListDisbursementsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateDisbursement: NewCreateDisbursementUseCase(createRepos, createServices),
		ReadDisbursement:   NewReadDisbursementUseCase(readRepos, readServices),
		UpdateDisbursement: NewUpdateDisbursementUseCase(updateRepos, updateServices),
		DeleteDisbursement: NewDeleteDisbursementUseCase(deleteRepos, deleteServices),
		ListDisbursements:  NewListDisbursementsUseCase(listRepos, listServices),
	}
}
