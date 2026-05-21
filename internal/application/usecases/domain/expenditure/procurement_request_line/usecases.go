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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadProcurementRequestLine: NewReadProcurementRequestLineUseCase(
			ReadProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			ReadProcurementRequestLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateProcurementRequestLine: NewUpdateProcurementRequestLineUseCase(
			UpdateProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			UpdateProcurementRequestLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteProcurementRequestLine: NewDeleteProcurementRequestLineUseCase(
			DeleteProcurementRequestLineRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			DeleteProcurementRequestLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListProcurementRequestLines: NewListProcurementRequestLinesUseCase(
			ListProcurementRequestLinesRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			ListProcurementRequestLinesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		GetProcurementRequestLineListPageData: NewGetProcurementRequestLineListPageDataUseCase(
			GetProcurementRequestLineListPageDataRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			GetProcurementRequestLineListPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		GetProcurementRequestLineItemPageData: NewGetProcurementRequestLineItemPageDataUseCase(
			GetProcurementRequestLineItemPageDataRepositories{ProcurementRequestLine: repositories.ProcurementRequestLine},
			GetProcurementRequestLineItemPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
