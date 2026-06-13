package suppliercontract

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// SupplierContractRepositories groups all repository dependencies for supplier contract use cases.
type SupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// SupplierContractServices groups all business service dependencies for supplier contract use cases.
type SupplierContractServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all supplier-contract-related use cases.
type UseCases struct {
	CreateSupplierContract          *CreateSupplierContractUseCase
	ReadSupplierContract            *ReadSupplierContractUseCase
	UpdateSupplierContract          *UpdateSupplierContractUseCase
	DeleteSupplierContract          *DeleteSupplierContractUseCase
	ListSupplierContracts           *ListSupplierContractsUseCase
	GetSupplierContractListPageData *GetSupplierContractListPageDataUseCase
	GetSupplierContractItemPageData *GetSupplierContractItemPageDataUseCase
	ApproveSupplierContract         *ApproveSupplierContractUseCase
	TerminateSupplierContract       *TerminateSupplierContractUseCase
	RegisterRelease                 *RegisterReleaseUseCase
	RegisterBilling                 *RegisterBillingUseCase
	RegisterCredit                  *RegisterCreditUseCase
}

// NewUseCases creates a new collection of supplier contract use cases.
func NewUseCases(
	repositories SupplierContractRepositories,
	services SupplierContractServices,
) *UseCases {
	return &UseCases{
		CreateSupplierContract: NewCreateSupplierContractUseCase(
			CreateSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			CreateSupplierContractServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSupplierContract: NewReadSupplierContractUseCase(
			ReadSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			ReadSupplierContractServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateSupplierContract: NewUpdateSupplierContractUseCase(
			UpdateSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			UpdateSupplierContractServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteSupplierContract: NewDeleteSupplierContractUseCase(
			DeleteSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			DeleteSupplierContractServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListSupplierContracts: NewListSupplierContractsUseCase(
			ListSupplierContractsRepositories{SupplierContract: repositories.SupplierContract},
			ListSupplierContractsServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		GetSupplierContractListPageData: NewGetSupplierContractListPageDataUseCase(
			GetSupplierContractListPageDataRepositories{SupplierContract: repositories.SupplierContract},
			GetSupplierContractListPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		GetSupplierContractItemPageData: NewGetSupplierContractItemPageDataUseCase(
			GetSupplierContractItemPageDataRepositories{SupplierContract: repositories.SupplierContract},
			GetSupplierContractItemPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ApproveSupplierContract: NewApproveSupplierContractUseCase(
			ApproveSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			ApproveSupplierContractServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		TerminateSupplierContract: NewTerminateSupplierContractUseCase(
			TerminateSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			TerminateSupplierContractServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		RegisterRelease: NewRegisterReleaseUseCase(
			RegisterReleaseRepositories{SupplierContract: repositories.SupplierContract},
			RegisterReleaseServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		RegisterBilling: NewRegisterBillingUseCase(
			RegisterBillingRepositories{SupplierContract: repositories.SupplierContract},
			RegisterBillingServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		RegisterCredit: NewRegisterCreditUseCase(
			RegisterCreditRepositories{SupplierContract: repositories.SupplierContract},
			RegisterCreditServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
