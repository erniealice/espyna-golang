package work_request

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	work_requestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// UseCases contains all work-request-related use cases.
type UseCases struct {
	CreateWorkRequest          *CreateWorkRequestUseCase
	ReadWorkRequest            *ReadWorkRequestUseCase
	UpdateWorkRequest          *UpdateWorkRequestUseCase
	ListWorkRequests           *ListWorkRequestsUseCase
	SubmitWorkRequest          *SubmitWorkRequestUseCase
	UpdateWorkRequestStatus    *UpdateWorkRequestStatusUseCase
	AssignWorkRequest          *AssignWorkRequestUseCase
	GetWorkRequestListPageData *GetWorkRequestListPageDataUseCase
	GetWorkRequestItemPageData *GetWorkRequestItemPageDataUseCase
	StampRequestSLABreaches    *StampRequestSLABreachesUseCase
	GetRequestInboxKPIs        *GetRequestInboxKPIsUseCase
}

// WorkRequestRepositories groups all repository dependencies for work request use cases.
type WorkRequestRepositories struct {
	WorkRequest     work_requestpb.WorkRequestDomainServiceServer
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer
	WorkspaceUser   workspaceuserpb.WorkspaceUserDomainServiceServer // FK validation for assignment
}

// WorkRequestServices groups all business service dependencies.
type WorkRequestServices struct {
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
}

// NewUseCases wires all work request use cases.
func NewUseCases(repositories WorkRequestRepositories, services WorkRequestServices) *UseCases {
	return &UseCases{
		CreateWorkRequest: NewCreateWorkRequestUseCase(
			CreateWorkRequestRepositories{WorkRequest: repositories.WorkRequest, WorkRequestType: repositories.WorkRequestType},
			CreateWorkRequestServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator, IDGenerator: services.IDGenerator},
		),
		ReadWorkRequest: NewReadWorkRequestUseCase(
			ReadWorkRequestRepositories{WorkRequest: repositories.WorkRequest},
			ReadWorkRequestServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		UpdateWorkRequest: NewUpdateWorkRequestUseCase(
			UpdateWorkRequestRepositories{WorkRequest: repositories.WorkRequest},
			UpdateWorkRequestServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		ListWorkRequests: NewListWorkRequestsUseCase(
			ListWorkRequestsRepositories{WorkRequest: repositories.WorkRequest},
			ListWorkRequestsServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		SubmitWorkRequest: NewSubmitWorkRequestUseCase(
			SubmitWorkRequestRepositories{WorkRequest: repositories.WorkRequest, WorkRequestType: repositories.WorkRequestType},
			SubmitWorkRequestServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		UpdateWorkRequestStatus: NewUpdateWorkRequestStatusUseCase(
			UpdateWorkRequestStatusRepositories{WorkRequest: repositories.WorkRequest},
			UpdateWorkRequestStatusServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		AssignWorkRequest: NewAssignWorkRequestUseCase(
			AssignWorkRequestRepositories{WorkRequest: repositories.WorkRequest, WorkspaceUser: repositories.WorkspaceUser},
			AssignWorkRequestServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		GetWorkRequestListPageData: NewGetWorkRequestListPageDataUseCase(
			ListWorkRequestsRepositories{WorkRequest: repositories.WorkRequest},
			ListWorkRequestsServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		GetWorkRequestItemPageData: NewGetWorkRequestItemPageDataUseCase(
			ListWorkRequestsRepositories{WorkRequest: repositories.WorkRequest},
			ListWorkRequestsServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		StampRequestSLABreaches: NewStampRequestSLABreachesUseCase(
			StampRequestSLABreachesRepositories{WorkRequest: repositories.WorkRequest},
			StampRequestSLABreachesServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
		GetRequestInboxKPIs: NewGetRequestInboxKPIsUseCase(
			GetRequestInboxKPIsRepositories{WorkRequest: repositories.WorkRequest},
			GetRequestInboxKPIsServices{ActionGatekeeper: services.ActionGatekeeper, Transactor: services.Transactor, Translator: services.Translator},
		),
	}
}
