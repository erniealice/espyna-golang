package revenue

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	recoverydocumentlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document_line"
)

// C32: the S1 aggregates are nil unless their own repository is wired.
func TestKnownCostRecoveryAggregatesAreNilWithoutTheirRepositories(t *testing.T) {
	gate := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	uc := NewUseCases(RevenueRepositories{}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate)
	if uc.DocumentSeries != nil || uc.RecoveryDocument != nil || uc.RecoveryDocumentLine != nil {
		t.Fatalf("S1 aggregates must be nil without repositories: %+v %+v %+v", uc.DocumentSeries, uc.RecoveryDocument, uc.RecoveryDocumentLine)
	}
	uc = NewUseCases(RevenueRepositories{
		DocumentSeries:       documentseriespb.UnimplementedDocumentSeriesDomainServiceServer{},
		RecoveryDocument:     recoverydocumentpb.UnimplementedRecoveryDocumentDomainServiceServer{},
		RecoveryDocumentLine: recoverydocumentlinepb.UnimplementedRecoveryDocumentLineDomainServiceServer{},
	}, ports.NewNoOpAuthorizer(), nil, ports.NewNoOpTranslator(), nil, gate)
	if uc.DocumentSeries == nil || uc.RecoveryDocument == nil || uc.RecoveryDocument.ListRecoverablesAging == nil || uc.RecoveryDocumentLine == nil {
		t.Fatal("S1 aggregates must be built when their repositories are wired")
	}
}
