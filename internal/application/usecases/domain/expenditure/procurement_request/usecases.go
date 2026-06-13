package procurementrequest

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// ProcurementRequestRepositories groups all repository dependencies.
//
// The CRIT-3 spawn saga (see approve.go) requires line-level access
// and the spawn dispatcher abstraction; both are owned by the sister
// agent's wiring but flow through this struct so the saga remains
// the single mutator of approval state.
type ProcurementRequestRepositories struct {
	ProcurementRequest     procurementrequestpb.ProcurementRequestDomainServiceServer
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
	SpawnDispatcher        SpawnDispatcher
}

// ProcurementRequestServices groups all business service dependencies.
//
// `ApprovalPolicyResolver` is HIGH-5 plumbing — nil falls back to
// `DefaultRequireApprovalResolver` (fail-safe).
type ProcurementRequestServices struct {
	Authorizer             ports.Authorizer
	Transactor             ports.Transactor
	Translator             ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator            ports.IDGenerator
	ApprovalPolicyResolver ApprovalPolicyResolver
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadProcurementRequest: NewReadProcurementRequestUseCase(
			ReadProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			ReadProcurementRequestServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateProcurementRequest: NewUpdateProcurementRequestUseCase(
			UpdateProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			UpdateProcurementRequestServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteProcurementRequest: NewDeleteProcurementRequestUseCase(
			DeleteProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			DeleteProcurementRequestServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListProcurementRequests: NewListProcurementRequestsUseCase(
			ListProcurementRequestsRepositories{ProcurementRequest: repositories.ProcurementRequest},
			ListProcurementRequestsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		GetProcurementRequestListPageData: NewGetProcurementRequestListPageDataUseCase(
			GetProcurementRequestListPageDataRepositories{ProcurementRequest: repositories.ProcurementRequest},
			GetProcurementRequestListPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		GetProcurementRequestItemPageData: NewGetProcurementRequestItemPageDataUseCase(
			GetProcurementRequestItemPageDataRepositories{ProcurementRequest: repositories.ProcurementRequest},
			GetProcurementRequestItemPageDataServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		SubmitProcurementRequest: NewSubmitProcurementRequestUseCase(
			SubmitProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			SubmitProcurementRequestServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ApproveProcurementRequest: NewApproveProcurementRequestUseCase(
			ApproveProcurementRequestRepositories{
				ProcurementRequest:     repositories.ProcurementRequest,
				ProcurementRequestLine: repositories.ProcurementRequestLine,
				SpawnDispatcher:        repositories.SpawnDispatcher,
			},
			ApproveProcurementRequestServices{
				Authorizer:             services.Authorizer,
				Transactor:             services.Transactor,
				Translator:             services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator:            services.IDGenerator,
				ApprovalPolicyResolver: services.ApprovalPolicyResolver,
			},
		),
		RejectProcurementRequest: NewRejectProcurementRequestUseCase(
			RejectProcurementRequestRepositories{ProcurementRequest: repositories.ProcurementRequest},
			RejectProcurementRequestServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		SpawnPurchaseOrder: NewSpawnPurchaseOrderUseCase(
			SpawnPurchaseOrderRepositories{ProcurementRequest: repositories.ProcurementRequest},
			SpawnPurchaseOrderServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
