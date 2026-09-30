package collection_application

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_effect"
	"github.com/erniealice/espyna-golang/registry/entityid"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// ReverseCollectionApplicationRepositories groups repository dependencies.
type ReverseCollectionApplicationRepositories struct {
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
	// Revenue locks a REVENUE target before its application is reversed (A1 m2).
	Revenue              revenuepb.RevenueDomainServiceServer
	RecoveryDocument     recoverydocumentpb.RecoveryDocumentDomainServiceServer
	RecoveryDocumentLine recoverydocumentlinepb.RecoveryDocumentLineDomainServiceServer
	BillableCharge       billablechargepb.BillableChargeDomainServiceServer
	ChargeComponent      chargecomponentpb.ChargeComponentDomainServiceServer
	ChargePolicyPosting  postingpb.ChargePolicyPostingDomainServiceServer
	ChargeEffect         chargeeffectpb.ChargeEffectDomainServiceServer
}

// ReverseCollectionApplicationServices groups service dependencies.
type ReverseCollectionApplicationServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	IDGenerator      ports.IDGenerator
}

// ReverseCollectionApplicationUseCase reverses an APPLIED application (D10): the original becomes
// REVERSED and a reversing row (reverses_application_id = original, status REVERSED) is written;
// neither counts in any balance again, so the target's balance is restored and the receipt's
// leftover grows. Recovery-document applications also write REVERSAL effects. The receipt itself is
// never changed. Permission: collection_application:reverse, strict gate (C3).
type ReverseCollectionApplicationUseCase struct {
	repositories ReverseCollectionApplicationRepositories
	services     ReverseCollectionApplicationServices
}

// NewReverseCollectionApplicationUseCase creates the use case with grouped dependencies.
func NewReverseCollectionApplicationUseCase(r ReverseCollectionApplicationRepositories, s ReverseCollectionApplicationServices) *ReverseCollectionApplicationUseCase {
	return &ReverseCollectionApplicationUseCase{repositories: r, services: s}
}

// Execute reverses the application; the response carries the reversal row.
func (uc *ReverseCollectionApplicationUseCase) Execute(ctx context.Context, req *collectionapplicationpb.ReverseCollectionApplicationRequest) (*collectionapplicationpb.ReverseCollectionApplicationResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionReverse); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "collection_application", err)
	}
	return &collectionapplicationpb.ReverseCollectionApplicationResponse{Data: out, Success: true}, nil
}

func (uc *ReverseCollectionApplicationUseCase) execute(ctx context.Context, req *collectionapplicationpb.ReverseCollectionApplicationRequest) (*collectionapplicationpb.CollectionApplication, error) {
	rp := uc.repositories
	if rp.CollectionApplication == nil || rp.RecoveryDocument == nil || rp.RecoveryDocumentLine == nil || rp.BillableCharge == nil ||
		rp.ChargeComponent == nil || rp.ChargePolicyPosting == nil || rp.ChargeEffect == nil {
		return nil, fmt.Errorf("collection_application: repositories not wired")
	}
	r := ledger{CollectionApplication: rp.CollectionApplication, Revenue: rp.Revenue, RecoveryDocument: rp.RecoveryDocument, RecoveryDocumentLine: rp.RecoveryDocumentLine,
		BillableCharge: rp.BillableCharge, ChargeComponent: rp.ChargeComponent, ChargePolicyPosting: rp.ChargePolicyPosting, ChargeEffect: rp.ChargeEffect}
	if req == nil || blank(req.GetCollectionApplicationId()) {
		return nil, errNotFound
	}
	applicationID := req.GetCollectionApplicationId()
	actor, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)

	read := func(tx context.Context) (*collectionapplicationpb.CollectionApplication, error) {
		resp, err := r.CollectionApplication.ReadCollectionApplication(tx, &collectionapplicationpb.ReadCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{Id: applicationID}})
		if err != nil && !usecaseerr.IsNotFound(err) {
			return nil, usecaseerr.RepoErr("collection_application", "read collection_application", err, applicationID)
		}
		if err != nil || len(resp.GetData()) == 0 {
			return nil, errNotFound
		}
		return resp.Data[0], nil
	}

	var out *collectionapplicationpb.CollectionApplication
	err = inTx(ctx, uc.services.Transactor, func(tx context.Context) error {
		// Lock the application row itself first (fail closed): two concurrent reversals of the same
		// application serialise here, and the loser re-reads it as already reversed.
		if err := r.lockApplication(tx, applicationID); err != nil {
			return err
		}
		orig, err := read(tx)
		if err != nil {
			return err
		}
		// Lock the target (document or invoice), then re-read the application: a reversal changes the
		// target's open balance, so it serialises with a concurrent receipt's lock on the same target
		// (A1 m2) and with a void of the document.
		switch orig.GetTargetKind() {
		case collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT:
			if err := r.lockDocument(tx, orig.GetRecoveryDocumentId()); err != nil {
				return err
			}
		case collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE:
			if err := r.lockRevenue(tx, orig.GetRevenueId()); err != nil {
				return err
			}
		}
		if orig, err = read(tx); err != nil {
			return err
		}
		if orig.GetStatus() != collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED || orig.GetReversesApplicationId() != "" {
			return errAlreadyReversed
		}
		now := time.Now()
		revID, err := newID(uc.services.IDGenerator)
		if err != nil {
			return err
		}
		if _, err := r.CollectionApplication.UpdateCollectionApplication(tx, &collectionapplicationpb.UpdateCollectionApplicationRequest{Data: &collectionapplicationpb.CollectionApplication{
			Id: orig.GetId(), Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED, DateModified: i64p(now.UnixMilli()),
		}}); err != nil {
			return usecaseerr.RepoErr("collection_application", "mark collection_application reversed", err, orig.GetId())
		}
		rev := &collectionapplicationpb.CollectionApplication{
			Id: revID, TreasuryCollectionId: orig.GetTreasuryCollectionId(), ClientId: orig.GetClientId(),
			TargetKind: orig.GetTargetKind(), ApplicationKind: orig.GetApplicationKind(),
			Amount: orig.GetAmount(), Currency: orig.GetCurrency(), AppliedAt: i64p(now.UnixMilli()), AppliedBy: strp(actor),
			OrderRank: orig.OrderRank, ReversesApplicationId: strp(orig.GetId()),
			Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED, Active: true,
			DateCreated: i64p(now.UnixMilli()), DateModified: i64p(now.UnixMilli()),
		}
		if workspaceID != "" {
			rev.WorkspaceId = workspaceID
		}
		if orig.RevenueId != nil {
			rev.RevenueId = orig.RevenueId
		}
		if orig.RecoveryDocumentId != nil {
			rev.RecoveryDocumentId = orig.RecoveryDocumentId
		}
		if orig.WithholdingCertificateId != nil {
			rev.WithholdingCertificateId = orig.WithholdingCertificateId
		}
		if orig.ChargeComponentId != nil {
			rev.ChargeComponentId = orig.ChargeComponentId
		}
		cr, err := r.CollectionApplication.CreateCollectionApplication(tx, &collectionapplicationpb.CreateCollectionApplicationRequest{Data: rev})
		if err != nil {
			return usecaseerr.RepoErr("collection_application", "create reversal collection_application", err, revID)
		}
		out = rev
		if len(cr.GetData()) > 0 {
			out = cr.Data[0]
		}

		if orig.GetTargetKind() == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT {
			version, err := r.versionOfDocument(tx, orig.GetRecoveryDocumentId())
			if err != nil {
				return err
			}
			number := ""
			if dr, err := r.RecoveryDocument.ReadRecoveryDocument(tx, &recoverydocumentpb.ReadRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: orig.GetRecoveryDocumentId()}}); err == nil && len(dr.GetData()) > 0 {
				number = dr.Data[0].GetDocumentNumber()
			}
			if err := charge_effect.RecordReversalEffects(tx, r.effectRepos(uc.services.IDGenerator, uc.services.Translator), version,
				enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, out.GetId(), today(), orig.GetCurrency(), orig.GetAmount(), number); err != nil {
				return mapEffectErr(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
