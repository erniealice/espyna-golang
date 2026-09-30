package recovery_document

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_effect"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// VoidRecoveryDocumentRepositories groups repository dependencies.
type VoidRecoveryDocumentRepositories struct {
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine  recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	BillableCharge        billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent       chargecomponentpb.ChargeComponentDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect          chargeeffectpb.ChargeEffectDomainServiceServer
}

// VoidRecoveryDocumentServices groups service dependencies.
type VoidRecoveryDocumentServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	IDGenerator      ports.IDGenerator
}

// VoidRecoveryDocumentUseCase voids an ISSUED document (row retained, never deleted - N19). It is
// refused while any APPLIED collection application targets it (has_applications) or while an ISSUED
// credit note corrects it (has_credit_notes: the credit would otherwise float free of any statement).
// The document becomes VOID, its charges CANCELLED, and the ISSUE effects are mirrored by REVERSAL
// effects (event id = the document id). The document row is locked first, so a concurrent apply
// (which locks the same row) or credit-note issuance serialises against it. Permission
// recovery_document:void, strict gate (C3).
type VoidRecoveryDocumentUseCase struct {
	repositories VoidRecoveryDocumentRepositories
	services     VoidRecoveryDocumentServices
}

// NewVoidRecoveryDocumentUseCase creates the use case with grouped dependencies.
func NewVoidRecoveryDocumentUseCase(r VoidRecoveryDocumentRepositories, s VoidRecoveryDocumentServices) *VoidRecoveryDocumentUseCase {
	return &VoidRecoveryDocumentUseCase{repositories: r, services: s}
}

// Execute voids the document.
func (uc *VoidRecoveryDocumentUseCase) Execute(ctx context.Context, req *recoverydocumentpb.VoidRecoveryDocumentRequest) (*recoverydocumentpb.VoidRecoveryDocumentResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionVoid); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", err)
	}
	return out, nil
}

func (uc *VoidRecoveryDocumentUseCase) execute(ctx context.Context, req *recoverydocumentpb.VoidRecoveryDocumentRequest) (*recoverydocumentpb.VoidRecoveryDocumentResponse, error) {
	r := uc.repositories
	if r.RecoveryDocument == nil || r.RecoveryDocumentLine == nil || r.BillableCharge == nil || r.ChargeComponent == nil ||
		r.CollectionApplication == nil || r.ChargePolicyPosting == nil || r.ChargeEffect == nil {
		return nil, fmt.Errorf("recovery_document: void repositories unavailable")
	}
	if req == nil || blank(req.GetRecoveryDocumentId()) {
		return nil, errNotFound
	}
	if blank(req.GetReason()) {
		return nil, errVoidReasonRequired
	}
	if uc.services.IDGenerator == nil {
		return nil, fmt.Errorf("recovery_document: id generator unavailable")
	}
	deps := documentDeps{RecoveryDocument: r.RecoveryDocument, RecoveryDocumentLine: r.RecoveryDocumentLine, BillableCharge: r.BillableCharge,
		ChargeComponent: r.ChargeComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargeEffect: r.ChargeEffect}

	var out *recoverydocumentpb.RecoveryDocument
	err := inTx(ctx, uc.services.Transactor, func(tx context.Context) error {
		doc, err := lockDocument(tx, r.RecoveryDocument, req.GetRecoveryDocumentId())
		if err != nil {
			return err
		}
		if doc.GetStatus() == recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_VOID {
			return errAlreadyVoid
		}
		apps, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*collectionapplicationpb.CollectionApplication, error) {
			resp, err := r.CollectionApplication.ListCollectionApplications(tx, &collectionapplicationpb.ListCollectionApplicationsRequest{Filters: listdata.EqFilter("recovery_document_id", doc.GetId()), Sort: s, Pagination: p})
			return resp.GetData(), err
		})
		if err != nil {
			return usecaseerr.RepoErr("recovery_document", "list collection_applications", err, doc.GetId())
		}
		for _, a := range apps {
			if a.GetStatus() == collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED {
				return errHasApplications
			}
		}
		notes, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*recoverydocumentpb.RecoveryDocument, error) {
			resp, err := r.RecoveryDocument.ListRecoveryDocuments(tx, &recoverydocumentpb.ListRecoveryDocumentsRequest{Filters: listdata.EqFilter("corrects_document_id", doc.GetId()), Sort: s, Pagination: p})
			return resp.GetData(), err
		})
		if err != nil {
			return usecaseerr.RepoErr("recovery_document", "list credit notes", err, doc.GetId())
		}
		for _, n := range notes {
			if n.GetStatus() == recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED {
				return errHasCreditNotes
			}
		}

		// Charges of the document, ascending id.
		lines, err := linesOf(tx, r.RecoveryDocumentLine, doc.GetId())
		if err != nil {
			return err
		}
		chargeIDs := map[string]bool{}
		for _, ln := range lines {
			cr, err := r.ChargeComponent.ReadChargeComponent(tx, &chargecomponentpb.ReadChargeComponentRequest{Data: &chargecomponentpb.ChargeComponent{Id: ln.GetChargeComponentId()}})
			if err != nil && !usecaseerr.IsNotFound(err) {
				return usecaseerr.RepoErr("recovery_document", "read charge_component", err, ln.GetChargeComponentId())
			}
			if err != nil || len(cr.GetData()) == 0 {
				return errNotFound
			}
			chargeIDs[cr.Data[0].GetBillableChargeId()] = true
		}
		ordered := make([]string, 0, len(chargeIDs))
		for id := range chargeIDs {
			ordered = append(ordered, id)
		}
		sort.Strings(ordered)
		version := ""
		var charges []*billablechargepb.BillableCharge
		for _, id := range ordered {
			c, err := deps.lockCharge(tx, id)
			if err != nil {
				return err
			}
			if v := c.GetChargePolicyVersionId(); version == "" {
				version = v
			} else if v != version {
				return errMissingPosting // a document never mixes versions (issuance groups by version)
			}
			charges = append(charges, c)
		}
		if blank(version) {
			return errMissingPosting
		}

		if err := charge_effect.RecordReversalEffects(tx, deps.effectRepos(uc.services.IDGenerator.GenerateID, uc.services.Translator), version, enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_ISSUE,
			doc.GetId(), time.Now().UTC().Format("2006-01-02"), doc.GetCurrency(), doc.GetTotalAmount(), doc.GetDocumentNumber()); err != nil {
			if isMissingPosting(err) {
				return errMissingPosting
			}
			return err
		}
		now := time.Now().UnixMilli()
		for _, c := range charges {
			if _, err := r.BillableCharge.UpdateBillableCharge(tx, &billablechargepb.UpdateBillableChargeRequest{Data: &billablechargepb.BillableCharge{
				Id: c.GetId(), Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_CANCELLED, DateModified: i64p(now),
			}}); err != nil {
				return usecaseerr.RepoErr("recovery_document", "cancel billable_charge", err, c.GetId())
			}
		}
		up, err := r.RecoveryDocument.UpdateRecoveryDocument(tx, &recoverydocumentpb.UpdateRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{
			Id: doc.GetId(), Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_VOID,
			VoidReason: strp(req.GetReason()), VoidedAt: i64p(now), DateModified: i64p(now),
		}})
		if err != nil {
			return usecaseerr.RepoErr("recovery_document", "void recovery_document", err, doc.GetId())
		}
		out = doc
		if len(up.GetData()) > 0 {
			out = up.Data[0]
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &recoverydocumentpb.VoidRecoveryDocumentResponse{Data: out, Success: true}, nil
}
