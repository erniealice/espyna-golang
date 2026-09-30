// Package collection_application holds every use case of the collection_application entity (domain
// treasury; 20260927-usage-and-pass-through-charges, Slice B S1, build-spec §6.2/§6.3/§6.4, §7c C27):
// the read use cases plus the CollectionApplicationDomainService behaviour RPCs
// ReceiveAndApplyCollection, PreviewCollectionApplication and ReverseCollectionApplication.
// collection_application rows are immutable financial records: the adapter refuses Delete (C6), a
// reversal writes a new row. Every Execute takes and returns esqyma proto messages (C1).
// Authorization: collection_application:{list,read} (fail closed); create/reverse use the strict gate.
package collection_application

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// Repositories groups every (workspace-scoped) repository the use cases read and write (aggregate
// input; each use case receives only the subset in its own <Op>Repositories). The cross-domain
// collaborators (revenue, recovery documents, charges, client) are passed in by the treasury
// initializer.
type Repositories struct {
	Collection            collectionpb.CollectionDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	Revenue               revenuepb.RevenueDomainServiceServer
	RevenuePayment        revenuepaymentpb.RevenuePaymentDomainServiceServer
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine  recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	BillableCharge        billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent       chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect          chargeeffectpb.ChargeEffectDomainServiceServer
	// Reference readers (C5): the request's client and collection method are read in the actor's
	// workspace before use. Client is required; CollectionMethod only when the request names one.
	Client           clientpb.ClientDomainServiceServer
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer
}

// Services groups the shared service dependencies (aggregate input).
type Services struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// Transactor and IDGenerator are required by the behaviour use cases (receive/reverse).
	Transactor  ports.Transactor
	IDGenerator ports.IDGenerator
}

// UseCases aggregates the collection_application use cases.
type UseCases struct {
	ListCollectionApplications           *ListCollectionApplicationsUseCase
	GetCollectionApplicationListPageData *GetCollectionApplicationListPageDataUseCase
	ReadCollectionApplication            *ReadCollectionApplicationUseCase
	ReceiveAndApplyCollection            *ReceiveAndApplyCollectionUseCase
	PreviewCollectionApplication         *PreviewCollectionApplicationUseCase
	ReverseCollectionApplication         *ReverseCollectionApplicationUseCase
}

// NewUseCases wires the collection_application use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		ListCollectionApplications: NewListCollectionApplicationsUseCase(
			ListCollectionApplicationsRepositories{CollectionApplication: r.CollectionApplication},
			ListCollectionApplicationsServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetCollectionApplicationListPageData: NewGetCollectionApplicationListPageDataUseCase(
			GetCollectionApplicationListPageDataRepositories{CollectionApplication: r.CollectionApplication},
			GetCollectionApplicationListPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ReadCollectionApplication: NewReadCollectionApplicationUseCase(
			ReadCollectionApplicationRepositories{CollectionApplication: r.CollectionApplication},
			ReadCollectionApplicationServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ReceiveAndApplyCollection: NewReceiveAndApplyCollectionUseCase(
			ReceiveAndApplyCollectionRepositories(r),
			ReceiveAndApplyCollectionServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, Transactor: s.Transactor, IDGenerator: s.IDGenerator}),
		PreviewCollectionApplication: NewPreviewCollectionApplicationUseCase(
			PreviewCollectionApplicationRepositories{
				Collection: r.Collection, CollectionApplication: r.CollectionApplication, Revenue: r.Revenue, RevenuePayment: r.RevenuePayment, RecoveryDocument: r.RecoveryDocument,
				Client: r.Client, CollectionMethod: r.CollectionMethod,
			},
			PreviewCollectionApplicationServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		ReverseCollectionApplication: NewReverseCollectionApplicationUseCase(
			ReverseCollectionApplicationRepositories{
				CollectionApplication: r.CollectionApplication, Revenue: r.Revenue, RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine,
				BillableCharge: r.BillableCharge, ChargeComponent: r.ChargeComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargeEffect: r.ChargeEffect,
			},
			ReverseCollectionApplicationServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper, Transactor: s.Transactor, IDGenerator: s.IDGenerator}),
	}
}
