package product

import (
	"context"
	"sort"
	"time"

	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productdashpb "github.com/erniealice/esqyma/pkg/schema/v1/service/dashboard/product"
)

// ProductDashboardRepository is the slice of the postgres product adapter the
// dashboard use case consumes. The postgres adapter `PostgresProductRepository`
// satisfies it — see
// `contrib/postgres/internal/adapter/product/product_dashboard_assertions.go`
// for the compile-time guarantee.
//
// Extension interface — the aggregate methods live on the postgres product
// adapter; this package surfaces them as a Go interface the composition root
// assembles via type assertion.
type ProductDashboardRepository interface {
	// CountByStatusAndKind returns {active: n, inactive: m} for the given
	// product_kind. Workspace-scoped.
	CountByStatusAndKind(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	// CountByLine returns line_id → count for active products of the given
	// product_kind. Workspace-scoped.
	CountByLine(ctx context.Context, workspaceID string, kind string) (map[string]int64, error)
	// RecentlyListed returns the most-recently-created active products of
	// the given product_kind, newest-first. Workspace-scoped.
	RecentlyListed(ctx context.Context, workspaceID string, kind string, limit int32) ([]*productpb.Product, error)
}

// GetProductDashboardRepositories groups the per-repository dependencies the
// service-layer product dashboard composes. The single repository may be nil
// when the postgres build tag is inactive — the Execute method tolerates nil
// and returns a zero-valued response.
type GetProductDashboardRepositories struct {
	Product ProductDashboardRepository
}

// GetProductDashboardUseCase composes the product aggregate into the
// service-layer dashboard projection.
//
// Per Q-SDM-DASHBOARD-LAYOUT (LOCKED 2026-05-20) and Q-SDM-DASHBOARD-SHARED-TYPES
// (LOCKED 2026-05-20), this use case owns the product-dashboard repository
// composition that previously lived at `usecases/product/dashboard/`. The
// relocation moves the proto contract out of the Go-only Request/Response
// shape and into the service-driven category, where it sits alongside the
// other dashboard candidates.
//
// **No authcheck.Check.** Per hexagonal-rules.md §8 service-driven domains
// take a conditional subset of layers; dashboard reads are authenticated by
// the upstream HTTP view middleware. Matches the Admin/Equity pilot pattern.
type GetProductDashboardUseCase struct {
	repositories GetProductDashboardRepositories
}

// NewGetProductDashboardUseCase wires the use case from grouped dependencies.
func NewGetProductDashboardUseCase(
	repositories GetProductDashboardRepositories,
) *GetProductDashboardUseCase {
	return &GetProductDashboardUseCase{repositories: repositories}
}

// Execute assembles the product dashboard proto response. The use case
// defaults Kind to "service" (matching the centymo service-mount surface)
// but accepts other kinds (e.g. "good") via the request. Failures degrade
// gracefully — a nil product adapter returns a zero-valued response.
//
// Per the codex review pattern (Schedule pilot P1 2026-05-20), the
// LineLabels/LineValues parallel slices come from a Go map. The order is
// sort-by-count-desc (so the chart shows largest lines first); ties are
// then broken alphabetically by line id so the proto output stays
// deterministic for cross-language clients.
func (uc *GetProductDashboardUseCase) Execute(
	ctx context.Context,
	req *productdashpb.GetProductDashboardRequest,
) (*productdashpb.GetProductDashboardResponse, error) {
	_ = time.Now() // reserved for future time-range filtering (NowMillis)
	workspaceID := ""
	kind := "service"
	if req != nil {
		workspaceID = req.GetWorkspaceId()
		if req.GetKind() != "" {
			kind = req.GetKind()
		}
	}

	resp := &productdashpb.GetProductDashboardResponse{
		Success: true,
		Stats:   &productdashpb.ProductStats{},
	}
	if uc.repositories.Product == nil {
		return resp, nil
	}

	// Active count.
	if byStatus, err := uc.repositories.Product.CountByStatusAndKind(ctx, workspaceID, kind); err == nil {
		resp.Stats.TotalActive = byStatus["active"]
	}

	// By-line breakdown for the chart — sort by count desc, then by key asc
	// to keep proto output deterministic across language clients.
	if byLine, err := uc.repositories.Product.CountByLine(ctx, workspaceID, kind); err == nil {
		type kv struct {
			key string
			val int64
		}
		rows := make([]kv, 0, len(byLine))
		for k, v := range byLine {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].val != rows[j].val {
				return rows[i].val > rows[j].val
			}
			return rows[i].key < rows[j].key
		})

		resp.LineLabels = make([]string, 0, len(rows))
		resp.LineValues = make([]float64, 0, len(rows))
		for _, r := range rows {
			resp.LineLabels = append(resp.LineLabels, r.key)
			resp.LineValues = append(resp.LineValues, float64(r.val))
		}
		resp.Stats.LineCount = int64(len(byLine))
	}

	// Recent additions (limit 5). Also used as top-revenue stand-in until a
	// line-item revenue join lands.
	if recent, err := uc.repositories.Product.RecentlyListed(ctx, workspaceID, kind, 5); err == nil {
		resp.Recent = recent
		resp.Stats.RecentlyAddedCnt = int64(len(recent))

		// Populate TopRevenue as top-by-recency placeholder.
		resp.TopRevenue = make([]*productdashpb.TopRevenueRow, 0, len(recent))
		for _, p := range recent {
			resp.TopRevenue = append(resp.TopRevenue, &productdashpb.TopRevenueRow{
				ProductId:   p.GetId(),
				ProductName: p.GetName(),
				Total:       0, // centavos — 0 until revenue join
			})
		}
		if len(recent) > 0 {
			resp.Stats.TopRevenueName = recent[0].GetName()
		}
	}

	return resp, nil
}
