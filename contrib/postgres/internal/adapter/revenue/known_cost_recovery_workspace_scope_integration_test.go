//go:build postgresql

package revenue

import (
	"context"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// DocumentSeries: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestDocumentSeriesWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "document_series")
	repo := NewPostgresDocumentSeriesRepository(h.Ops, entityid.DocumentSeries).(*PostgresDocumentSeriesRepository)
	const id = "s1scope-document_series"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateDocumentSeries(ctxA, &documentseriespb.CreateDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, Code: "SCOPE", IssuerName: "Issuer", DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, FiscalReset: documentseriespb.DocumentSeriesFiscalReset_DOCUMENT_SERIES_FISCAL_RESET_NONE, NextNumber: 1, NumberPadding: 6, Status: documentseriespb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE, Name: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadDocumentSeries(ctxA, &documentseriespb.ReadDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadDocumentSeries(ctx, &documentseriespb.ReadDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetDocumentSeriesItemPageData(ctx, &documentseriespb.GetDocumentSeriesItemPageDataRequest{DocumentSeriesId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateDocumentSeries(ctx, &documentseriespb.UpdateDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{Id: id, Name: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteDocumentSeries(ctx, &documentseriespb.DeleteDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListDocumentSeries(ctx, &documentseriespb.ListDocumentSeriesRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetDocumentSeriesListPageData(ctx, &documentseriespb.GetDocumentSeriesListPageDataRequest{}); err == nil {
				for _, r := range l.DocumentSeriesList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
			if _, err := repo.LockDocumentSeriesForUpdate(ctx, id); err == nil {
				t.Errorf("%s: LockDocumentSeriesForUpdate must report not found", name)
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadDocumentSeries(ctxA, &documentseriespb.ReadDocumentSeriesRequest{Data: &documentseriespb.DocumentSeries{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetName() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListDocumentSeries(ctxA, &documentseriespb.ListDocumentSeriesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
		if v, err := repo.LockDocumentSeriesForUpdate(ctxA, id); err != nil || v.GetId() != id {
			t.Errorf("own LockDocumentSeriesForUpdate: %v", err)
		}
	})
	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	if _, err := repo.LockDocumentSeriesForUpdate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: scopetest.WsA}), id); err == nil {
		t.Error("LockDocumentSeriesForUpdate outside a transaction must fail closed")
	}
}

// RecoveryDocument: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestRecoveryDocumentWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "recovery_document")
	repo := NewPostgresRecoveryDocumentRepository(h.Ops, entityid.RecoveryDocument).(*PostgresRecoveryDocumentRepository)
	const id = "s1scope-recovery_document"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateRecoveryDocument(ctxA, &recoverydocumentpb.CreateRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, DocumentSeriesId: "ds1", SequenceNumber: 1, DocumentNumber: "SCOPE-000001", DocumentType: recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_STATEMENT, ClientId: "cl1", TotalAmount: 1, Currency: "PHP", Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED, IssuanceKey: "scope-key", VoidReason: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadRecoveryDocument(ctxA, &recoverydocumentpb.ReadRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadRecoveryDocument(ctx, &recoverydocumentpb.ReadRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetRecoveryDocumentItemPageData(ctx, &recoverydocumentpb.GetRecoveryDocumentItemPageDataRequest{RecoveryDocumentId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateRecoveryDocument(ctx, &recoverydocumentpb.UpdateRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id, VoidReason: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteRecoveryDocument(ctx, &recoverydocumentpb.DeleteRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListRecoveryDocuments(ctx, &recoverydocumentpb.ListRecoveryDocumentsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetRecoveryDocumentListPageData(ctx, &recoverydocumentpb.GetRecoveryDocumentListPageDataRequest{}); err == nil {
				for _, r := range l.RecoveryDocumentList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
			if _, err := repo.LockRecoveryDocumentForUpdate(ctx, id); err == nil {
				t.Errorf("%s: LockRecoveryDocumentForUpdate must report not found", name)
			}
		}
		// C6: the row is an immutable financial record - even the owning workspace cannot delete it.
		if _, err := repo.DeleteRecoveryDocument(ctxA, &recoverydocumentpb.DeleteRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id}}); err == nil {
			t.Error("own-workspace delete must be refused (immutable)")
		} else if !postgresCore.IsImmutableRecord(err) {
			t.Errorf("own-workspace delete must be an immutable_record refusal, got %v", err)
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadRecoveryDocument(ctxA, &recoverydocumentpb.ReadRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetVoidReason() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListRecoveryDocuments(ctxA, &recoverydocumentpb.ListRecoveryDocumentsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
		if v, err := repo.LockRecoveryDocumentForUpdate(ctxA, id); err != nil || v.GetId() != id {
			t.Errorf("own LockRecoveryDocumentForUpdate: %v", err)
		}
	})
	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	if _, err := repo.LockRecoveryDocumentForUpdate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: scopetest.WsA}), id); err == nil {
		t.Error("LockRecoveryDocumentForUpdate outside a transaction must fail closed")
	}
}

// RecoveryDocumentLine: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestRecoveryDocumentLineWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "recovery_document_line")
	repo := NewPostgresRecoveryDocumentLineRepository(h.Ops, entityid.RecoveryDocumentLine).(*PostgresRecoveryDocumentLineRepository)
	const id = "s1scope-recovery_document_line"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateRecoveryDocumentLine(ctxA, &recoverydocumentlinepb.CreateRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, RecoveryDocumentId: "rd1", ChargeComponentId: "cc1", Amount: 1, Currency: "PHP", Description: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadRecoveryDocumentLine(ctxA, &recoverydocumentlinepb.ReadRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadRecoveryDocumentLine(ctx, &recoverydocumentlinepb.ReadRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetRecoveryDocumentLineItemPageData(ctx, &recoverydocumentlinepb.GetRecoveryDocumentLineItemPageDataRequest{RecoveryDocumentLineId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateRecoveryDocumentLine(ctx, &recoverydocumentlinepb.UpdateRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id, Description: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteRecoveryDocumentLine(ctx, &recoverydocumentlinepb.DeleteRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListRecoveryDocumentLines(ctx, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetRecoveryDocumentLineListPageData(ctx, &recoverydocumentlinepb.GetRecoveryDocumentLineListPageDataRequest{}); err == nil {
				for _, r := range l.RecoveryDocumentLineList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// C6: the row is an immutable financial record - even the owning workspace cannot delete it.
		if _, err := repo.DeleteRecoveryDocumentLine(ctxA, &recoverydocumentlinepb.DeleteRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id}}); err == nil {
			t.Error("own-workspace delete must be refused (immutable)")
		} else if !postgresCore.IsImmutableRecord(err) {
			t.Errorf("own-workspace delete must be an immutable_record refusal, got %v", err)
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadRecoveryDocumentLine(ctxA, &recoverydocumentlinepb.ReadRecoveryDocumentLineRequest{Data: &recoverydocumentlinepb.RecoveryDocumentLine{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetDescription() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListRecoveryDocumentLines(ctxA, &recoverydocumentlinepb.ListRecoveryDocumentLinesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}
