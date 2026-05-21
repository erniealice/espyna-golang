package procurementrequestline

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// ProcurementRequestLineRepositories groups all repository dependencies.
type ProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// ProcurementRequestLineServices groups all business service dependencies.
type ProcurementRequestLineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all procurement request line use cases.
type UseCases struct {
	CreateProcurementRequestLine          *CreateProcurementRequestLineUseCase
	ReadProcurementRequestLine            *ReadProcurementRequestLineUseCase
	UpdateProcurementRequestLine          *UpdateProcurementRequestLineUseCase
	DeleteProcurementRequestLine          *DeleteProcurementRequestLineUseCase
	ListProcurementRequestLines           *ListProcurementRequestLinesUseCase
	GetProcurementRequestLineListPageData *GetProcurementRequestLineListPageDataUseCase
	GetProcurementRequestLineItemPageData *GetProcurementRequestLineItemPageDataUseCase
}

// NewUseCases creates a new collection of procurement request line use cases.
func NewUseCases(
	repositories ProcurementRequestLineRepositories,
	services ProcurementRequestLineServices,
) *UseCases {
	return &UseCases{
		CreateProcurementRequestLine: NewCreateProcurementRequestLineUseCase(
			CreateProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			CreateProcurementRequestLineServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadProcurementRequestLine: NewReadProcurementRequestLineUseCase(
			ReadProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			ReadProcurementRequestLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateProcurementRequestLine: NewUpdateProcurementRequestLineUseCase(
			UpdateProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			UpdateProcurementRequestLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteProcurementRequestLine: NewDeleteProcurementRequestLineUseCase(
			DeleteProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			DeleteProcurementRequestLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListProcurementRequestLines: NewListProcurementRequestLinesUseCase(
			ListProcurementRequestLinesRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			ListProcurementRequestLinesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetProcurementRequestLineListPageData: NewGetProcurementRequestLineListPageDataUseCase(
			GetProcurementRequestLineListPageDataRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			GetProcurementRequestLineListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		GetProcurementRequestLineItemPageData: NewGetProcurementRequestLineItemPageDataUseCase(
			GetProcurementRequestLineItemPageDataRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			GetProcurementRequestLineItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
