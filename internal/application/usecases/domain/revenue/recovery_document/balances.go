package recovery_document

// Recovery balance helpers (build-spec §7c C27): the settlement position of ISSUED recovery documents
// is owned by the recovery_document entity. LoadSnapshot is the ONE balance rule (C24); the treasury
// collection_application use cases read open items through it as a collaborator, so the
// "Receive & apply" preview shows exactly the balance the reports show.

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

// BalanceRepos are the repositories the balance helpers read.
type BalanceRepos struct {
	RecoveryDocument      recoverydocumentpb.RecoveryDocumentDomainServiceServer
	CollectionApplication collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// DocumentBalance is the settlement position of one ISSUED STATEMENT recovery document.
type DocumentBalance struct {
	Document *recoverydocumentpb.RecoveryDocument
	// Credited is the signed (<= 0) sum of ISSUED credit notes that correct this document.
	Credited int64
	// Applied is the sum of APPLIED CASH applications targeting this document.
	Applied int64
	// Balance = total + credited - applied (may be negative after a correction of a paid document).
	Balance int64
}

// Snapshot is the recovery position of a client (or of the whole workspace when the client id is
// blank) as of a date.
type Snapshot struct {
	Statements  []DocumentBalance
	CreditNotes []*recoverydocumentpb.RecoveryDocument
	// Applications are the APPLIED CASH applications that target recovery documents.
	Applications []*collectionapplicationpb.CollectionApplication
}

const dateLayout = "2006-01-02"

func appliedDate(a *collectionapplicationpb.CollectionApplication) string {
	if a.GetAppliedAt() == 0 {
		return ""
	}
	return time.UnixMilli(a.GetAppliedAt()).UTC().Format(dateLayout)
}

// verifyClient reads the request-supplied client through the row-scoped ReadClient before any list
// runs (C17, mirrors collection_application verifyReferences): a client outside the caller's row scope or
// workspace reads as not found. It fails closed when the reader is not wired.
func verifyClient(ctx context.Context, repo clientpb.ClientDomainServiceServer, clientID string) error {
	if repo == nil {
		return fmt.Errorf("recovery_document: client repository not wired")
	}
	cr, err := repo.ReadClient(ctx, &clientpb.ReadClientRequest{Data: &clientpb.Client{Id: clientID}})
	if err != nil && !usecaseerr.IsNotFound(err) {
		return usecaseerr.RepoErr("recovery_document", "read client", err, clientID)
	}
	if err != nil || len(cr.GetData()) == 0 {
		return errNotFound
	}
	return nil
}

// LoadSnapshot reads the recovery documents and cash applications of clientID (blank = every client
// of the caller's workspace; the repositories enforce the workspace) and derives each statement's
// balance. asOf (ISO date, blank = now) excludes documents issued after it and applications made
// after it. VOID documents and REVERSED applications never count.
func LoadSnapshot(ctx context.Context, repos BalanceRepos, clientID, asOf string) (*Snapshot, error) {
	if repos.RecoveryDocument == nil || repos.CollectionApplication == nil {
		return nil, fmt.Errorf("recovery_document: balance repositories not wired")
	}
	var filter *commonpb.FilterRequest
	if !blank(clientID) {
		filter = listdata.EqFilter("client_id", clientID)
	}
	docs, err := listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*recoverydocumentpb.RecoveryDocument, error) {
		r, err := repos.RecoveryDocument.ListRecoveryDocuments(ctx, &recoverydocumentpb.ListRecoveryDocumentsRequest{Filters: filter, Sort: sort, Pagination: p})
		return r.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list documents", err)
	}
	apps, err := listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*collectionapplicationpb.CollectionApplication, error) {
		r, err := repos.CollectionApplication.ListCollectionApplications(ctx, &collectionapplicationpb.ListCollectionApplicationsRequest{Filters: filter, Sort: sort, Pagination: p})
		return r.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("recovery_document", "list applications", err)
	}

	snap := &Snapshot{}
	applied := map[string]int64{}
	for _, a := range apps {
		if !a.GetActive() ||
			a.GetStatus() != collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED ||
			a.GetApplicationKind() != collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH ||
			a.GetTargetKind() != collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT ||
			a.GetRecoveryDocumentId() == "" {
			continue
		}
		if asOf != "" {
			if d := appliedDate(a); d != "" && d > asOf {
				continue
			}
		}
		snap.Applications = append(snap.Applications, a)
		applied[a.GetRecoveryDocumentId()] += a.GetAmount()
	}

	credits := map[string]int64{}
	var statements []*recoverydocumentpb.RecoveryDocument
	for _, d := range docs {
		if !d.GetActive() || d.GetStatus() != recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED {
			continue
		}
		if asOf != "" && d.GetIssueDate() != "" && d.GetIssueDate() > asOf {
			continue
		}
		if d.GetDocumentType() == recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE {
			snap.CreditNotes = append(snap.CreditNotes, d)
			credits[d.GetCorrectsDocumentId()] += d.GetTotalAmount()
			continue
		}
		statements = append(statements, d)
	}
	sort.Slice(statements, func(i, j int) bool { return statements[i].GetId() < statements[j].GetId() })
	for _, d := range statements {
		b := DocumentBalance{Document: d, Credited: credits[d.GetId()], Applied: applied[d.GetId()]}
		b.Balance = d.GetTotalAmount() + b.Credited - b.Applied
		snap.Statements = append(snap.Statements, b)
	}
	return snap, nil
}
