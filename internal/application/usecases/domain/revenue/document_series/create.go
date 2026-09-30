package document_series

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// CreateDocumentSeriesRepositories groups repository dependencies.
type CreateDocumentSeriesRepositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// CreateDocumentSeriesServices groups service dependencies.
type CreateDocumentSeriesServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// CreateDocumentSeriesUseCase creates an ACTIVE series. next_number starts at 1 unless the
// caller supplies a positive starting number (migrating an existing numbering sequence).
type CreateDocumentSeriesUseCase struct {
	repositories CreateDocumentSeriesRepositories
	services     CreateDocumentSeriesServices
}

// NewCreateDocumentSeriesUseCase creates the use case with grouped dependencies.
func NewCreateDocumentSeriesUseCase(r CreateDocumentSeriesRepositories, s CreateDocumentSeriesServices) *CreateDocumentSeriesUseCase {
	return &CreateDocumentSeriesUseCase{repositories: r, services: s}
}

func (uc *CreateDocumentSeriesUseCase) codeExists(ctx context.Context, code string) (bool, error) {
	resp, err := uc.repositories.DocumentSeries.ListDocumentSeries(ctx, &pb.ListDocumentSeriesRequest{Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "code",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: code, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}}}})
	if err != nil {
		return false, usecaseerr.RepoErr("document_series", "look up series code", err)
	}
	return len(resp.GetData()) > 0, nil
}

// Execute creates the series.
func (uc *CreateDocumentSeriesUseCase) Execute(ctx context.Context, req *pb.CreateDocumentSeriesRequest) (*pb.CreateDocumentSeriesResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionCreate); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "document_series", err)
	}
	return out, nil
}

func (uc *CreateDocumentSeriesUseCase) execute(ctx context.Context, req *pb.CreateDocumentSeriesRequest) (*pb.CreateDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil {
		return nil, invalid("data is required")
	}
	d := req.Data
	if !codePattern.MatchString(d.GetCode()) {
		return nil, invalid("code must be UPPER_CASE letters, digits, '_' or '-'")
	}
	if blank(d.GetIssuerName()) {
		return nil, invalid("issuer_name is required")
	}
	if d.GetDocumentKind() != enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT &&
		d.GetDocumentKind() != enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE {
		return nil, invalid("document_kind is required")
	}
	reset := d.GetFiscalReset()
	if reset == pb.DocumentSeriesFiscalReset_DOCUMENT_SERIES_FISCAL_RESET_UNSPECIFIED {
		reset = pb.DocumentSeriesFiscalReset_DOCUMENT_SERIES_FISCAL_RESET_NONE
	}
	next := d.GetNextNumber()
	if next < 0 {
		return nil, invalid("next_number must be positive")
	}
	if next == 0 {
		next = 1
	}
	// number_padding is required and >= 1: the column is NOT NULL DEFAULT 6 and the storage bridge
	// drops a zero scalar, so 0 would silently become 6 (build-spec C7; no proto change, R4 m5).
	padding := d.GetNumberPadding()
	if padding < 1 || padding > 18 {
		return nil, invalid("number_padding must be between 1 and 18")
	}
	if uc.services.IDGenerator == nil {
		return nil, invalid("id generator unavailable")
	}
	if uc.repositories.DocumentSeries == nil {
		return nil, repoUnavailable()
	}
	taken, err := uc.codeExists(ctx, d.GetCode())
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errCodeTaken
	}
	ms, ts := stamp()
	row := &pb.DocumentSeries{
		Id: uc.services.IDGenerator.GenerateID(), Code: d.GetCode(), Name: d.Name,
		IssuerName: d.GetIssuerName(), IssuerTaxId: d.IssuerTaxId,
		DocumentKind: d.GetDocumentKind(), Prefix: d.Prefix, BranchCode: d.BranchCode,
		FiscalReset: reset, NextNumber: next, NumberPadding: padding,
		Status: pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE,
		Active: true, DateCreated: i64p(ms), DateCreatedString: strp(ts), DateModified: i64p(ms), DateModifiedString: strp(ts),
	}
	resp, err := uc.repositories.DocumentSeries.CreateDocumentSeries(ctx, &pb.CreateDocumentSeriesRequest{Data: row})
	if err != nil {
		return nil, usecaseerr.RepoErr("document_series", "create document_series", err, row.Id)
	}
	return resp, nil
}
