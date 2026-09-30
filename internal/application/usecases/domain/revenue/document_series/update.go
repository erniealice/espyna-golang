package document_series

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

// UpdateDocumentSeriesRepositories groups repository dependencies.
type UpdateDocumentSeriesRepositories struct {
	DocumentSeries pb.DocumentSeriesDomainServiceServer
}

// UpdateDocumentSeriesServices groups service dependencies.
type UpdateDocumentSeriesServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	Transactor       ports.Transactor
}

// UpdateDocumentSeriesUseCase edits descriptive fields and may retire a series. code,
// document_kind, next_number and fiscal_reset are immutable here (numbering integrity); a RETIRED
// series cannot be edited or reactivated (retire only). The series row is locked FOR UPDATE first
// (fail closed: a repository that cannot lock refuses the update), so an edit or retirement
// serialises against a concurrent issuance.
type UpdateDocumentSeriesUseCase struct {
	repositories UpdateDocumentSeriesRepositories
	services     UpdateDocumentSeriesServices
}

// NewUpdateDocumentSeriesUseCase creates the use case with grouped dependencies.
func NewUpdateDocumentSeriesUseCase(r UpdateDocumentSeriesRepositories, s UpdateDocumentSeriesServices) *UpdateDocumentSeriesUseCase {
	return &UpdateDocumentSeriesUseCase{repositories: r, services: s}
}

// Execute updates or retires the series.
func (uc *UpdateDocumentSeriesUseCase) Execute(ctx context.Context, req *pb.UpdateDocumentSeriesRequest) (*pb.UpdateDocumentSeriesResponse, error) {
	if err := gateStrict(ctx, uc.services.ActionGatekeeper, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "document_series", err)
	}
	return out, nil
}

func (uc *UpdateDocumentSeriesUseCase) execute(ctx context.Context, req *pb.UpdateDocumentSeriesRequest) (*pb.UpdateDocumentSeriesResponse, error) {
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, invalid("id is required")
	}
	if uc.repositories.DocumentSeries == nil {
		return nil, repoUnavailable()
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errTransactionRequired
	}
	d := req.Data
	if d.GetIssuerName() != "" && blank(d.GetIssuerName()) {
		return nil, invalid("issuer_name must not be blank")
	}
	// number_padding is optional (partial update, like the other S1 status verbs): 0 means "not
	// sent, keep the stored value" (the storage bridge drops a zero anyway); a sent value is 1..18.
	if d.GetNumberPadding() != 0 && (d.GetNumberPadding() < 1 || d.GetNumberPadding() > 18) {
		return nil, invalid("number_padding must be between 1 and 18")
	}
	var out *pb.UpdateDocumentSeriesResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		locker, ok := uc.repositories.DocumentSeries.(domainports.DocumentSeriesLocker)
		if !ok {
			return fmt.Errorf("document_series: repository cannot lock rows")
		}
		cur, err := locker.LockDocumentSeriesForUpdate(txCtx, d.GetId())
		switch {
		case err != nil && usecaseerr.IsNotFound(err):
			return errNotFound
		case err != nil:
			return usecaseerr.RepoErr("document_series", "lock document_series", err, d.GetId())
		case cur == nil:
			return errNotFound
		}
		if cur.GetStatus() == pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED {
			return errRetired
		}
		if d.GetStatus() == pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE {
			return invalid("status ACTIVE cannot be set; only RETIRED")
		}
		// Once a number has been issued (next_number > 1) the prefix and padding are frozen.
		if cur.GetNextNumber() > 1 {
			if d.Prefix != nil && d.GetPrefix() != cur.GetPrefix() {
				return errNumberingLocked
			}
			if d.GetNumberPadding() != 0 && d.GetNumberPadding() != cur.GetNumberPadding() {
				return errNumberingLocked
			}
		}
		ms, ts := stamp()
		patch := &pb.DocumentSeries{
			Id: cur.GetId(), Name: d.Name, IssuerName: d.GetIssuerName(), IssuerTaxId: d.IssuerTaxId,
			Prefix: d.Prefix, BranchCode: d.BranchCode, NumberPadding: d.GetNumberPadding(),
			Status: d.GetStatus(), DateModified: i64p(ms), DateModifiedString: strp(ts),
		}
		out, err = uc.repositories.DocumentSeries.UpdateDocumentSeries(txCtx, &pb.UpdateDocumentSeriesRequest{Data: patch})
		if err != nil {
			return usecaseerr.RepoErr("document_series", "update document_series", err, cur.GetId())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
