// Package recovery_document holds the use cases of the recovery_document entity (domain revenue;
// 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.3/§6.4): the list/read
// use cases plus IssueRecoveryDocuments and VoidRecoveryDocument. Every Execute takes and returns
// esqyma proto messages (C1); named refusals are usecaseerr.Error values with ErrorCode() (C2); issue and void
// use the strict permission gate (C3). GetRecoveryBalance / ListRecoverablesAging are the read-only
// report RPCs of RecoveryDocumentDomainService (C27).
package recovery_document

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// Repositories groups every repository dependency (aggregate input; each use case receives only the
// subset in its own <Op>Repositories).
type Repositories struct {
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine  recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	DocumentSeries        documentseriespb.DocumentSeriesDomainServiceServer
	BillableCharge        billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent       chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect          chargeeffectpb.ChargeEffectDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	// Client is the row-scoped reference reader of the balance/aging reports (C17): a request-supplied
	// client id is read through it before any list runs.
	Client clientpb.ClientDomainServiceServer
}

// Services groups every shared service dependency (aggregate input).
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// Transactor and IDGenerator are required by the behaviour use cases (issue/void).
	Transactor  ports.Transactor
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the recovery_document use cases.
type UseCases struct {
	ListRecoveryDocuments           *ListRecoveryDocumentsUseCase
	GetRecoveryDocumentListPageData *GetRecoveryDocumentListPageDataUseCase
	ReadRecoveryDocument            *ReadRecoveryDocumentUseCase
	IssueRecoveryDocuments          *IssueRecoveryDocumentsUseCase
	VoidRecoveryDocument            *VoidRecoveryDocumentUseCase
	// RecoveryDocumentDomainService report RPCs (read only)
	GetRecoveryBalance    *GetRecoveryBalanceUseCase
	ListRecoverablesAging *ListRecoverablesAgingUseCase
}

// NewUseCases wires the recovery_document use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListRecoveryDocuments: NewListRecoveryDocumentsUseCase(
			ListRecoveryDocumentsRepositories{RecoveryDocument: r.RecoveryDocument},
			ListRecoveryDocumentsServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetRecoveryDocumentListPageData: NewGetRecoveryDocumentListPageDataUseCase(
			GetRecoveryDocumentListPageDataRepositories{RecoveryDocument: r.RecoveryDocument},
			GetRecoveryDocumentListPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ReadRecoveryDocument: NewReadRecoveryDocumentUseCase(
			ReadRecoveryDocumentRepositories{RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine},
			ReadRecoveryDocumentServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		IssueRecoveryDocuments: NewIssueRecoveryDocumentsUseCase(
			IssueRecoveryDocumentsRepositories{
				RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine, DocumentSeries: r.DocumentSeries,
				BillableCharge: r.BillableCharge, ChargeComponent: r.ChargeComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargeEffect: r.ChargeEffect,
			},
			IssueRecoveryDocumentsServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, Transactor: s.Transactor, IDGenerator: s.IDGenerator}),
		VoidRecoveryDocument: NewVoidRecoveryDocumentUseCase(
			VoidRecoveryDocumentRepositories{
				RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine, BillableCharge: r.BillableCharge,
				ChargeComponent: r.ChargeComponent, CollectionApplication: r.CollectionApplication, ChargePolicyPosting: r.ChargePolicyPosting, ChargeEffect: r.ChargeEffect,
			},
			VoidRecoveryDocumentServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, Transactor: s.Transactor, IDGenerator: s.IDGenerator}),
		GetRecoveryBalance: NewGetRecoveryBalanceUseCase(
			GetRecoveryBalanceRepositories{Balance: BalanceRepos{RecoveryDocument: r.RecoveryDocument, CollectionApplication: r.CollectionApplication}, Client: r.Client},
			GetRecoveryBalanceServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ListRecoverablesAging: NewListRecoverablesAgingUseCase(
			ListRecoverablesAgingRepositories{Balance: BalanceRepos{RecoveryDocument: r.RecoveryDocument, CollectionApplication: r.CollectionApplication}, Client: r.Client},
			ListRecoverablesAgingServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
	}
}
