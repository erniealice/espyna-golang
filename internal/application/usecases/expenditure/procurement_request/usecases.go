package procurementrequest

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// ProcurementRequestRepositories groups all repository dependencies.
type ProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// ProcurementRequestServices groups all business service dependencies.
type ProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all procurement-request-related use cases.
type UseCases struct {
	CreateProcurementRequest          *CreateProcurementRequestUseCase
	ReadProcurementRequest            *ReadProcurementRequestUseCase
	UpdateProcurementRequest          *UpdateProcurementRequestUseCase
	DeleteProcurementRequest          *DeleteProcurementRequestUseCase
	ListProcurementRequests           *ListProcurementRequestsUseCase
	GetProcurementRequestListPageData *GetProcurementRequestListPageDataUseCase
	GetProcurementRequestItemPageData *GetProcurementRequestItemPageDataUseCase
	SubmitProcurementRequest          *SubmitProcurementRequestUseCase
	ApproveProcurementRequest         *ApproveProcurementRequestUseCase
	RejectProcurementRequest          *RejectProcurementRequestUseCase
	SpawnPurchaseOrder                *SpawnPurchaseOrderUseCase
}

// NewUseCases creates a new collection of procurement request use cases.
func NewUseCases(
	repositories ProcurementRequestRepositories,
	services ProcurementRequestServices,
) *UseCases {
	return &UseCases{
		CreateProcurementRequest: NewCreateProcurementRequestUseCase(
			CreateProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			CreateProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadProcurementRequest: NewReadProcurementRequestUseCase(
			ReadProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			ReadProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateProcurementRequest: NewUpdateProcurementRequestUseCase(
			UpdateProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			UpdateProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteProcurementRequest: NewDeleteProcurementRequestUseCase(
			DeleteProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			DeleteProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListProcurementRequests: NewListProcurementRequestsUseCase(
			ListProcurementRequestsRepositories{ProcurementRequest: repositories.ProcurementRequest},
			ListProcurementRequestsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetProcurementRequestListPageData: NewGetProcurementRequestListPageDataUseCase(
			GetProcurementRequestListPageDataRepositories{ProcurementRequest: repositories.ProcurementRequest},
			GetProcurementRequestListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetProcurementRequestItemPageData: NewGetProcurementRequestItemPageDataUseCase(
			GetProcurementRequestItemPageDataRepositories{ProcurementRequest: repositories.ProcurementRequest},
			GetProcurementRequestItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		SubmitProcurementRequest: NewSubmitProcurementRequestUseCase(
			SubmitProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			SubmitProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ApproveProcurementRequest: NewApproveProcurementRequestUseCase(
			ApproveProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			ApproveProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		RejectProcurementRequest: NewRejectProcurementRequestUseCase(
			RejectProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			RejectProcurementRequestServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		SpawnPurchaseOrder: NewSpawnPurchaseOrderUseCase(
			SpawnPurchaseOrderRepositories{ProcurementRequest: repositories.ProcurementRequest},
			SpawnPurchaseOrderServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
