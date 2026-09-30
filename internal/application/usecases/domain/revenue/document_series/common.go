package document_series

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]*$`)

func gate(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.DocumentSeries, Action: action})
}

// gateStrict is the shadow-immune gate of the numbering writes (build-spec §7c C26, §10b): create
// starts a fiscal numbering trail and update carries the ACTIVE -> RETIRED status change, so neither
// may inherit shadow mode's allow-on-deny (precedent: charge_policy/retire_charge_policy.go).
func gateStrict(ctx context.Context, gk *actiongate.ActionGatekeeper, action string) error {
	return gk.CheckStrict(ctx, &actiongate.CheckActionRequest{Entity: entityid.DocumentSeries, Action: action})
}

func blank(s string) bool   { return strings.TrimSpace(s) == "" }
func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }
func stamp() (int64, string) {
	now := time.Now()
	return now.UnixMilli(), now.Format(time.RFC3339)
}

// readSeries reads one row. A row the repository reports as missing is not_found; any other
// repository failure is logged and returned, never reported as not_found (C9).
func readSeries(ctx context.Context, repo pb.DocumentSeriesDomainServiceServer, id string) (*pb.DocumentSeries, error) {
	if blank(id) {
		return nil, invalid("id is required")
	}
	if repo == nil {
		return nil, repoUnavailable()
	}
	resp, err := repo.ReadDocumentSeries(ctx, &pb.ReadDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id}})
	if err != nil && !usecaseerr.IsNotFound(err) {
		return nil, usecaseerr.RepoErr("document_series", "read document_series", err, id)
	}
	if err != nil || resp == nil || len(resp.Data) == 0 {
		return nil, errNotFound
	}
	return resp.Data[0], nil
}
