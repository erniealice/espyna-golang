package suppliercontract

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	suppliercontractpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract"
)

// SupplierContractRepositories groups all repository dependencies for supplier contract use cases.
type SupplierContractRepositories struct {
	SupplierContract suppliercontractpb.SupplierContractDomainServiceServer
}

// SupplierContractServices groups all business service dependencies for supplier contract use cases.
type SupplierContractServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadSupplierContract: NewReadSupplierContractUseCase(
			ReadSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			ReadSupplierContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateSupplierContract: NewUpdateSupplierContractUseCase(
			UpdateSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			UpdateSupplierContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteSupplierContract: NewDeleteSupplierContractUseCase(
			DeleteSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			DeleteSupplierContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		ListSupplierContracts: NewListSupplierContractsUseCase(
			ListSupplierContractsRepositories{SupplierContract: repositories.SupplierContract},
			ListSupplierContractsServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		GetSupplierContractListPageData: NewGetSupplierContractListPageDataUseCase(
			GetSupplierContractListPageDataRepositories{SupplierContract: repositories.SupplierContract},
			GetSupplierContractListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		GetSupplierContractItemPageData: NewGetSupplierContractItemPageDataUseCase(
			GetSupplierContractItemPageDataRepositories{SupplierContract: repositories.SupplierContract},
			GetSupplierContractItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		ApproveSupplierContract: NewApproveSupplierContractUseCase(
			ApproveSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			ApproveSupplierContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		TerminateSupplierContract: NewTerminateSupplierContractUseCase(
			TerminateSupplierContractRepositories{SupplierContract: repositories.SupplierContract},
			TerminateSupplierContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		RegisterRelease: NewRegisterReleaseUseCase(
			RegisterReleaseRepositories{SupplierContract: repositories.SupplierContract},
			RegisterReleaseServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		RegisterBilling: NewRegisterBillingUseCase(
			RegisterBillingRepositories{SupplierContract: repositories.SupplierContract},
			RegisterBillingServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		RegisterCredit: NewRegisterCreditUseCase(
			RegisterCreditRepositories{SupplierContract: repositories.SupplierContract},
			RegisterCreditServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
