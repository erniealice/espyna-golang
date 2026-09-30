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
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// ReceiveAndApplyCollectionRepositories groups repository dependencies.
type ReceiveAndApplyCollectionRepositories Repositories

// ReceiveAndApplyCollectionServices groups service dependencies.
type ReceiveAndApplyCollectionServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
	IDGenerator      ports.IDGenerator
}

// ReceiveAndApplyCollectionUseCase: one transaction creates the receipt (treasury_collection with
// revenue_id NULL, stamped with client/workspace/currency) and CASH applications against the
// client's open items in N9 order until the amount is exhausted; the leftover stays unapplied.
// APPLICATION effects (DR CASH / CR RECEIVABLE) are written for recovery-document applications only.
// Every open invoice and recovery document of the client is locked (ascending id) before the plan is
// built, so an application can never race a void or a concurrent receipt (C4/C16).
// Permission: collection_application:create, strict gate (C3).
type ReceiveAndApplyCollectionUseCase struct {
	repositories ReceiveAndApplyCollectionRepositories
	services     ReceiveAndApplyCollectionServices
}

// NewReceiveAndApplyCollectionUseCase creates the use case with grouped dependencies.
func NewReceiveAndApplyCollectionUseCase(r ReceiveAndApplyCollectionRepositories, s ReceiveAndApplyCollectionServices) *ReceiveAndApplyCollectionUseCase {
	return &ReceiveAndApplyCollectionUseCase{repositories: r, services: s}
}

// Execute receives the cash and applies it.
func (uc *ReceiveAndApplyCollectionUseCase) Execute(ctx context.Context, req *collectionapplicationpb.ReceiveAndApplyCollectionRequest) (*collectionapplicationpb.ReceiveAndApplyCollectionResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionCreate); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "collection_application", err)
	}
	return out, nil
}

func (uc *ReceiveAndApplyCollectionUseCase) execute(ctx context.Context, request *collectionapplicationpb.ReceiveAndApplyCollectionRequest) (*collectionapplicationpb.ReceiveAndApplyCollectionResponse, error) {
	req := paramsFromReceive(request)
	if err := validate(req); err != nil {
		return nil, err
	}
	l := ledger(uc.repositories)
	if !l.readWired() || l.RecoveryDocumentLine == nil || l.BillableCharge == nil || l.ChargeComponent == nil || l.ChargePolicyPosting == nil || l.ChargeEffect == nil {
		return nil, fmt.Errorf("collection_application: repositories not wired")
	}
	actor, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	workspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)

	var out *collectionapplicationpb.ReceiveAndApplyCollectionResponse
	err = inTx(ctx, uc.services.Transactor, func(tx context.Context) error {
		// Request-supplied reference ids are read in the actor's workspace before use (C5).
		if err := l.verifyReferences(tx, req); err != nil {
			return err
		}
		// Serialise against concurrent receipts, voids and reversals BEFORE the balances are read for
		// real: lock every open invoice and recovery document of the client (fail closed).
		if err := l.lockOpenItems(tx, req.ClientID); err != nil {
			return err
		}

		plan, err := l.buildPlan(tx, req)
		if err != nil {
			return err
		}

		payDate := req.PaymentDate
		if blank(payDate) {
			payDate = today()
		}
		now := time.Now()
		colID, err := newID(uc.services.IDGenerator)
		if err != nil {
			return err
		}
		name := req.Name
		if name == "" {
			name = "Receipt"
		}
		col := &collectionpb.Collection{
			Id: colID, Name: name, Amount: req.Amount, Currency: req.Currency, Status: "completed",
			PaymentDate: payDate, ReferenceNumber: req.ReferenceNumber, CollectionMethodId: req.CollectionMethodID,
			ReceivedBy: actor, CollectionType: ReceiptCollectionType, ClientId: strp(req.ClientID), Active: true,
			DateCreated: i64p(now.UnixMilli()), DateModified: i64p(now.UnixMilli()),
		}
		if workspaceID != "" {
			col.WorkspaceId = strp(workspaceID)
		}
		cr, err := l.Collection.CreateCollection(tx, &collectionpb.CreateCollectionRequest{Data: col})
		if err != nil {
			return usecaseerr.RepoErr("collection_application", "create treasury_collection", err, colID)
		}
		created := col
		if len(cr.GetData()) > 0 {
			created = cr.Data[0]
		}

		resp := &collectionapplicationpb.ReceiveAndApplyCollectionResponse{Collection: created, Plan: plan, Unapplied: plan.Unapplied, Success: true}
		for _, al := range plan.Allocations {
			appID, err := newID(uc.services.IDGenerator)
			if err != nil {
				return err
			}
			app := &collectionapplicationpb.CollectionApplication{
				Id: appID, TreasuryCollectionId: created.GetId(), ClientId: req.ClientID,
				TargetKind:      al.TargetKind,
				ApplicationKind: collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH,
				Amount:          al.Apply, Currency: req.Currency, AppliedAt: i64p(now.UnixMilli()), AppliedBy: strp(actor),
				OrderRank: i32p(al.Rank), Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED, Active: true,
				DateCreated: i64p(now.UnixMilli()), DateModified: i64p(now.UnixMilli()),
			}
			if workspaceID != "" {
				app.WorkspaceId = workspaceID
			}
			if al.TargetKind == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE {
				app.RevenueId = strp(al.TargetId)
			} else {
				app.RecoveryDocumentId = strp(al.TargetId)
			}
			ar, err := l.CollectionApplication.CreateCollectionApplication(tx, &collectionapplicationpb.CreateCollectionApplicationRequest{Data: app})
			if err != nil {
				return usecaseerr.RepoErr("collection_application", "create collection_application", err, appID)
			}
			if len(ar.GetData()) > 0 {
				app = ar.Data[0]
			}
			resp.Applications = append(resp.Applications, app)

			if al.TargetKind == collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT {
				version, err := l.versionOfDocument(tx, al.TargetId)
				if err != nil {
					return err
				}
				if err := charge_effect.RecordChargeEffects(tx, l.effectRepos(uc.services.IDGenerator, uc.services.Translator), version,
					enumspb.ChargePostingEvent_CHARGE_POSTING_EVENT_APPLICATION, app.GetId(), payDate, req.Currency, al.Apply, al.Number); err != nil {
					return mapEffectErr(err)
				}
			}
		}
		out = resp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
